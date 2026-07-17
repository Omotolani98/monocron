package reconcile

import (
	"context"
	"log/slog"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/Omotolani98/monocron/internal/runner/controllerclient"
	"github.com/Omotolani98/monocron/internal/runner/daemonclient"
	"github.com/google/uuid"
)

// Runner is the host agent reconciliation loop.
type Runner struct {
	controller *controllerclient.Client
	daemon     *daemonclient.Client
	log        *slog.Logger
	cfg        RunnerConfig
	stop       chan struct{}
}

// RunnerConfig configures the runner loop.
type RunnerConfig struct {
	RunnerID      uuid.UUID
	Version       string
	DaemonVersion string
	Labels        contracts.Labels
	PollInterval  time.Duration
	HeartbeatInterval time.Duration
}

// NewRunner creates a runner agent.
func NewRunner(controller *controllerclient.Client, daemon *daemonclient.Client, log *slog.Logger, cfg RunnerConfig) *Runner {
	return &Runner{
		controller: controller,
		daemon:     daemon,
		log:        log,
		cfg:        cfg,
		stop:       make(chan struct{}),
	}
}

// Start begins the reconciliation and heartbeat loops.
func (r *Runner) Start(ctx context.Context) {
	go r.heartbeatLoop(ctx)
	go r.pollLoop(ctx)
}

// Stop signals the runner to stop.
func (r *Runner) Stop() {
	close(r.stop)
}

func (r *Runner) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case <-ticker.C:
			r.sendHeartbeat(ctx)
		}
	}
}

func (r *Runner) sendHeartbeat(ctx context.Context) {
	req := contracts.HeartbeatRequest{
		Version:       r.cfg.Version,
		DaemonVersion: r.cfg.DaemonVersion,
		Labels:        r.cfg.Labels,
		Healthy:       r.daemonHealth(ctx),
		Assignments:   []string{},
	}
	resp, err := r.controller.Heartbeat(ctx, req)
	if err != nil {
		r.log.Error("heartbeat failed", "error", err)
		return
	}
	r.log.Debug("heartbeat accepted", "state", resp.State)
}

func (r *Runner) daemonHealth(ctx context.Context) bool {
	return r.daemon.Health(ctx) == nil
}

func (r *Runner) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()
	var generation int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case <-ticker.C:
			resp, err := r.controller.PollAssignments(ctx, generation)
			if err != nil {
				r.log.Error("poll assignments", "error", err)
				continue
			}
			if len(resp.Assignments) == 0 {
				continue
			}
			for _, a := range resp.Assignments {
				r.applyAssignment(ctx, a)
			}
			if resp.Generation > generation {
				generation = resp.Generation
			}
		}
	}
}

func (r *Runner) applyAssignment(ctx context.Context, a contracts.Assignment) {
	schedule, err := r.controller.GetSchedule(ctx, a.ScheduleID) // We need a client method for this.
	if err != nil {
		r.log.Error("fetch schedule", "schedule_id", a.ScheduleID, "error", err)
		r.reportObservation(ctx, a, contracts.AssignmentStateRejected, err.Error())
		return
	}
	if !schedule.Enabled {
		if err := r.daemon.DeleteAssignment(ctx, a.ID); err != nil {
			r.log.Error("delete disabled assignment", "assignment_id", a.ID, "error", err)
		}
		r.reportObservation(ctx, a, contracts.AssignmentStateRemoved, "")
		return
	}
	da := contracts.DaemonAssignment{
		ID:         a.ID,
		ScheduleID: a.ScheduleID,
		Name:       schedule.Name,
		Cron:       schedule.Cron,
		Timezone:   schedule.Timezone,
		Timeout:    schedule.Timeout,
		Command:    schedule.Command,
		Generation: a.Generation,
		Enabled:    schedule.Enabled,
	}
	if err := r.daemon.ApplyAssignment(ctx, da); err != nil {
		r.log.Error("apply assignment", "assignment_id", a.ID, "error", err)
		r.reportObservation(ctx, a, contracts.AssignmentStateRejected, err.Error())
		return
	}
	r.reportObservation(ctx, a, contracts.AssignmentStateApplied, "")
}

func (r *Runner) reportObservation(ctx context.Context, a contracts.Assignment, state contracts.AssignmentState, msg string) {
	obs := contracts.AssignmentObservation{
		AssignmentID: a.ID,
		Generation:   a.Generation,
		State:        state,
		ErrorMessage: msg,
	}
	if err := r.controller.SubmitObservations(ctx, []contracts.AssignmentObservation{obs}); err != nil {
		r.log.Error("submit observation", "error", err)
	}
}
