package contracts

import (
	"time"

	"github.com/google/uuid"
)

// Pagination is used for list responses.
type Pagination struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ErrorResponse) Error() string { return e.Message }

// ScheduleSpec defines a recurring job.
type ScheduleSpec struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Cron        string    `json:"cron"`
	Timezone    string    `json:"timezone,omitempty"`
	Timeout     string    `json:"timeout"`
	Command     []string  `json:"command"`
	Labels      Labels    `json:"labels,omitempty"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Labels is a map used for runner selection.
type Labels map[string]string

// RunnerType describes the kind of runner host.
type RunnerType string

const (
	RunnerTypeVM        RunnerType = "vm"
	RunnerTypeDocker    RunnerType = "docker"
	RunnerTypeBareMetal RunnerType = "bare_metal"
)

// RunnerState describes the lifecycle state of a runner.
type RunnerState string

const (
	RunnerStatePending  RunnerState = "pending"
	RunnerStateLive     RunnerState = "live"
	RunnerStateDraining RunnerState = "draining"
	RunnerStateOffline  RunnerState = "offline"
	RunnerStateRevoked  RunnerState = "revoked"
)

// Runner is a registered host.
type Runner struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	Type          RunnerType  `json:"type"`
	Labels        Labels      `json:"labels,omitempty"`
	State         RunnerState `json:"state"`
	Version       string      `json:"version,omitempty"`
	DaemonVersion string      `json:"daemon_version,omitempty"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	CreatedAt     time.Time   `json:"created_at"`
}

// AssignmentState describes where an assignment is in its lifecycle.
type AssignmentState string

const (
	AssignmentStatePending  AssignmentState = "pending"
	AssignmentStateApplied  AssignmentState = "applied"
	AssignmentStateRejected AssignmentState = "rejected"
	AssignmentStateRemoved  AssignmentState = "removed"
)

// Assignment is the placement of a schedule onto a runner.
type Assignment struct {
	ID         uuid.UUID       `json:"id"`
	ScheduleID uuid.UUID       `json:"schedule_id"`
	RunnerID   uuid.UUID       `json:"runner_id"`
	State      AssignmentState `json:"state"`
	Generation int64           `json:"generation"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// ExecutionState describes the state of a single run.
type ExecutionState string

const (
	ExecutionStateQueued    ExecutionState = "queued"
	ExecutionStateRunning   ExecutionState = "running"
	ExecutionStateSucceeded ExecutionState = "succeeded"
	ExecutionStateFailed    ExecutionState = "failed"
	ExecutionStateTimedOut  ExecutionState = "timed_out"
	ExecutionStateCanceled  ExecutionState = "canceled"
)

// Execution represents one run of an assignment.
type Execution struct {
	ID           uuid.UUID      `json:"id"`
	AssignmentID uuid.UUID      `json:"assignment_id"`
	ScheduleID   uuid.UUID      `json:"schedule_id"`
	RunnerID     uuid.UUID      `json:"runner_id"`
	State        ExecutionState `json:"state"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty"`
	ExitCode     *int           `json:"exit_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ExecutionEvent is a state change or progress event for an execution.
type ExecutionEvent struct {
	ID           uuid.UUID      `json:"id"`
	ExecutionID  uuid.UUID      `json:"execution_id"`
	AssignmentID uuid.UUID      `json:"assignment_id"`
	State        ExecutionState `json:"state"`
	Message      string         `json:"message,omitempty"`
	EmittedAt    time.Time      `json:"emitted_at"`
}

// LogChunk is a sequenced chunk of execution output.
type LogChunk struct {
	ID          uuid.UUID `json:"id"`
	ExecutionID uuid.UUID `json:"execution_id"`
	Stream      string    `json:"stream"`
	Sequence    int64     `json:"sequence"`
	Payload     []byte    `json:"payload"`
	EmittedAt   time.Time `json:"emitted_at"`
}

// AuditEvent records who did what and why.
type AuditEvent struct {
	ID        uuid.UUID `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Reason    string    `json:"reason,omitempty"`
	Details   []byte    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
