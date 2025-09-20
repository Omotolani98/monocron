package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	conn "github.com/Omotolani98/monocron-runner/db"
	"github.com/Omotolani98/monocron-runner/internal/db"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

type CmdJob struct {
	Name    string
	Command []string
	Timeout time.Duration
	Do      func(ctx context.Context, name string, argv []string, timeout time.Duration) error
}

func (c CmdJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	_ = c.Do(ctx, c.Name, c.Command, c.Timeout)
}

type CronManager struct {
	c    *cron.Cron
	mu   sync.RWMutex
	jobs map[string][]cron.EntryID
	DB   conn.DB
}

func NewCronManager(loc *time.Location, logger cron.Logger, db conn.DB) *CronManager {
	return &CronManager{
		c: cron.New(
			cron.WithLocation(loc),
			cron.WithLogger(logger),
			cron.WithSeconds(),
			cron.WithChain(
				cron.SkipIfStillRunning(logger),
				cron.Recover(logger),
			),
		),
		jobs: make(map[string][]cron.EntryID),
		DB:   db,
	}
}

func (m *CronManager) Start() { m.c.Start() }

func (m *CronManager) Stop(ctx context.Context) error {
	stopCtx := m.c.Stop()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func normalizeSpec(spec string) (string, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", errors.New("empty spec")
	}
	if strings.HasPrefix(s, "@") {
		return s, nil
	}

	fields := strings.Fields(s)
	switch len(fields) {
	case 5:
		return "0 " + s, nil
	case 6:
		return s, nil
	default:
		return "", fmt.Errorf("invalid cron spec: %q", s)
	}
}

func NextRun(spec string, now time.Time) (time.Time, error) {
	ns, err := normalizeSpec(spec)
	if err != nil {
		return time.Time{}, err
	}
	sched, err := parser.Parse(ns)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(now), nil
}

type AddResult struct {
	JobID    string
	EntryIDs []int
}

func (m *CronManager) AddOrReplace(jobID string, specs []string, job CmdJob) (*AddResult, error) {
	q := db.New(m.DB.Pool)
	log.Info("We are getting things ready for you :)")

	if jobID == "" {
		return nil, errors.New("jobID required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if ids, ok := m.jobs[jobID]; ok {
		for _, id := range ids {
			m.c.Remove(id)
		}
		delete(m.jobs, jobID)
		if err := q.DeleteJob(context.Background(), uuid.MustParse(jobID)); err != nil {
			log.Warnf("failed to delete old job %s: %v", jobID, err)
		}
	}

	var eidList []cron.EntryID
	for _, sp := range specs {
		ns, err := normalizeSpec(sp)
		if err != nil {
			return nil, err
		}

		id, err := m.c.AddJob(ns, job)
		if err != nil {
			return nil, fmt.Errorf("add %q: %w", ns, err)
		}
		eidList = append(eidList, id)

		next, err := NextRun(ns, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to compute next run: %w", err)
		}

		dbJob, err := q.AddJob(context.Background(), db.AddJobParams{
			ID:          uuid.MustParse(jobID),
			EntryID:     int(id),
			Name:        job.Name,
			Status:      "pending",
			CronSpec:    ns,
			ScheduledAt: next,
		})
		if err != nil {
			return nil, fmt.Errorf("persist job %s: %w", jobID, err)
		}
		log.Infof("Persisted job %s (%s) with entry_id=%d next=%s",
			dbJob.ID, dbJob.Name, dbJob.EntryID, dbJob.ScheduledAt)
	}

	m.jobs[jobID] = eidList

	intIDs := make([]int, len(eidList))
	for i, id := range eidList {
		intIDs[i] = int(id)
	}

	return &AddResult{JobID: jobID, EntryIDs: intIDs}, nil
}

func (m *CronManager) Remove(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids, ok := m.jobs[jobID]
	if !ok {
		return false
	}
	for _, id := range ids {
		m.c.Remove(id)
	}
	delete(m.jobs, jobID)
	return true
}

type JobSummary struct {
	JobID       string
	EntryID     int
	Name        string
	Status      string
	CronSpec    string
	Next        time.Time
	Prev        time.Time
	ScheduledAt time.Time
}

func (m *CronManager) List(ctx context.Context, limit, offset int32) ([]JobSummary, error) {
	q := db.New(m.DB.Pool)

	rows, err := q.ListJobs(ctx, db.ListJobsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("list jobs from db: %w", err)
	}

	m.mu.RLock()
	entryByID := make(map[int32]cron.Entry, len(m.jobs))
	for _, ids := range m.jobs {
		for _, id := range ids {
			e := m.c.Entry(id)
			// robfig EntryID is int; your DB column is int32
			entryByID[int32(id)] = e
		}
	}
	m.mu.RUnlock()

	out := make([]JobSummary, 0, len(rows))
	for _, r := range rows {
		var e cron.Entry
		out = append(out, JobSummary{
			JobID:       r.ID.String(),
			EntryID:     r.EntryID,
			Name:        r.Name,
			Status:      r.Status,
			CronSpec:    r.CronSpec,
			Next:        e.Next,
			Prev:        e.Prev,
			ScheduledAt: r.ScheduledAt,
		})
	}
	return out, nil
}
