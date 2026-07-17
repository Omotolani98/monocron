package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/daemon/executor"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Assignment mirrors the desired state the daemon must run locally.
type Assignment struct {
	ID         uuid.UUID
	ScheduleID uuid.UUID
	Name       string
	Cron       string
	Timezone   string
	Timeout    time.Duration
	Command    []string
	Generation int64
	Enabled    bool
}

// ExecutionRecord is a single run of an assignment.
type ExecutionRecord struct {
	ID           uuid.UUID
	AssignmentID uuid.UUID
	ScheduleID   uuid.UUID
	State        contracts.ExecutionState
	StartedAt    *time.Time
	FinishedAt   *time.Time
	ExitCode     *int
	ErrorMessage string
	Stdout       []byte
	Stderr       []byte
}

// Callbacks are invoked by the scheduler to report state and persist data.
type Callbacks interface {
	PersistExecution(ctx context.Context, rec ExecutionRecord) error
	AppendLogChunk(ctx context.Context, execID uuid.UUID, stream string, seq int64, payload []byte) error
}

// Scheduler wraps the robfig cron scheduler with assignment-level state.
type Scheduler struct {
	cron       *cron.Cron
	mu         sync.RWMutex
	entries    map[uuid.UUID]cron.EntryID
	assignments map[uuid.UUID]*Assignment
	history    map[uuid.UUID][]ExecutionRecord
	log        *slog.Logger
	cb         Callbacks
	logLimit   int
	maxHistory int
}

// New creates a scheduler.
func New(log *slog.Logger, cb Callbacks, logLimit int) *Scheduler {
	if logLimit <= 0 {
		logLimit = 256 * 1024
	}
	return &Scheduler{
		cron: cron.New(
			cron.WithSeconds(),
			cron.WithChain(
				cron.SkipIfStillRunning(cron.DefaultLogger),
				cron.Recover(cron.DefaultLogger),
			),
		),
		entries:     map[uuid.UUID]cron.EntryID{},
		assignments: map[uuid.UUID]*Assignment{},
		history:     map[uuid.UUID][]ExecutionRecord{},
		log:         log,
		cb:          cb,
		logLimit:    logLimit,
		maxHistory:  100,
	}
}

// Start begins the scheduler.
func (s *Scheduler) Start() { s.cron.Start() }

// Stop halts the scheduler and waits for running jobs to complete.
func (s *Scheduler) Stop() context.Context { return s.cron.Stop() }

// Add or update an assignment. The assignment is keyed by ID and generation.
func (s *Scheduler) Add(ctx context.Context, a Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.assignments[a.ID]
	if ok && existing.Generation >= a.Generation {
		return nil
	}

	spec := a.Cron
	if a.Timezone != "" {
		spec = "CRON_TZ=" + a.Timezone + " " + a.Cron
	}
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(spec); err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", a.Cron, err)
	}

	if oldID, ok := s.entries[a.ID]; ok {
		s.cron.Remove(oldID)
		delete(s.entries, a.ID)
	}

	if !a.Enabled {
		s.assignments[a.ID] = &a
		return nil
	}

	entryID, err := s.cron.AddFunc(spec, s.run(a))
	if err != nil {
		return fmt.Errorf("schedule assignment: %w", err)
	}
	s.entries[a.ID] = entryID
	s.assignments[a.ID] = &a
	return nil
}

// Remove deletes an assignment and stops its scheduled runs.
func (s *Scheduler) Remove(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
	delete(s.assignments, id)
	return nil
}

// Get returns an assignment by ID.
func (s *Scheduler) Get(id uuid.UUID) (Assignment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.assignments[id]
	if !ok {
		return Assignment{}, false
	}
	return *a, true
}

// List returns all active assignments.
func (s *Scheduler) List() []Assignment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Assignment, 0, len(s.assignments))
	for _, a := range s.assignments {
		out = append(out, *a)
	}
	return out
}

// Cancel attempts to cancel a running execution. Currently best-effort via context.
func (s *Scheduler) Cancel(executionID uuid.UUID) bool {
	// Cancellations are handled via per-execution context in future iterations.
	return false
}

func (s *Scheduler) run(a Assignment) func() {
	return func() {
		s.execute(a)
	}
}

func (s *Scheduler) execute(a Assignment) {
	execID := uuid.New()
	now := time.Now().UTC()
	rec := ExecutionRecord{
		ID:           execID,
		AssignmentID: a.ID,
		ScheduleID:   a.ScheduleID,
		State:        contracts.ExecutionStateRunning,
		StartedAt:    &now,
	}

	s.mu.Lock()
	s.history[a.ID] = append(s.history[a.ID], rec)
	if len(s.history[a.ID]) > s.maxHistory {
		s.history[a.ID] = s.history[a.ID][len(s.history[a.ID])-s.maxHistory:]
	}
	s.mu.Unlock()

	if s.cb != nil {
		if err := s.cb.PersistExecution(context.Background(), rec); err != nil {
			s.log.Error("persist execution start", "error", err)
		}
	}

	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	s.log.Info("execution started", "execution_id", execID, "assignment_id", a.ID, "schedule", a.Name)

	res := executor.Run(ctx, a.Command, s.logLimit)
	res.State = evaluateState(res.State, ctx.Err())

	finished := time.Now().UTC()
	rec.State = res.State
	rec.FinishedAt = &finished
	rec.ExitCode = res.ExitCode
	rec.ErrorMessage = res.ErrorMessage
	rec.Stdout = res.Stdout
	rec.Stderr = res.Stderr

	s.mu.Lock()
	for i := range s.history[a.ID] {
		if s.history[a.ID][i].ID == execID {
			s.history[a.ID][i] = rec
			break
		}
	}
	s.mu.Unlock()

	if s.cb != nil {
		if err := s.cb.PersistExecution(context.Background(), rec); err != nil {
			s.log.Error("persist execution end", "error", err)
		}
		seq := int64(1)
		if len(rec.Stdout) > 0 {
			if err := s.cb.AppendLogChunk(context.Background(), execID, "stdout", seq, rec.Stdout); err != nil {
				s.log.Error("persist stdout", "error", err)
			}
			seq++
		}
		if len(rec.Stderr) > 0 {
			if err := s.cb.AppendLogChunk(context.Background(), execID, "stderr", seq, rec.Stderr); err != nil {
				s.log.Error("persist stderr", "error", err)
			}
		}
	}

	s.log.Info("execution finished", "execution_id", execID, "assignment_id", a.ID, "state", rec.State)
}

func evaluateState(state contracts.ExecutionState, ctxErr error) contracts.ExecutionState {
	if state == contracts.ExecutionStateFailed && ctxErr != nil {
		return contracts.ExecutionStateTimedOut
	}
	return state
}
