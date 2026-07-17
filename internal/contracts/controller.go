package contracts

import (
	"time"

	"github.com/google/uuid"
)

// Controller API request/response types.

type CreateScheduleRequest struct {
	Name        string   `json:"name"`
	Cron        string   `json:"cron"`
	Timezone    string   `json:"timezone,omitempty"`
	Timeout     string   `json:"timeout"`
	Command     []string `json:"command"`
	Labels      Labels   `json:"labels,omitempty"`
	Description string   `json:"description,omitempty"`
}

type UpdateScheduleRequest struct {
	Name        string   `json:"name,omitempty"`
	Cron        string   `json:"cron,omitempty"`
	Timezone    string   `json:"timezone,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	Command     []string `json:"command,omitempty"`
	Labels      Labels   `json:"labels,omitempty"`
	Description string   `json:"description,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

type ListSchedulesResponse struct {
	Items      []ScheduleSpec `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type CreateEnrollmentTokenRequest struct {
	Scope   string     `json:"scope,omitempty"`
	Labels  Labels     `json:"labels,omitempty"`
	Expires *time.Time `json:"expires,omitempty"`
}

type CreateEnrollmentTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ListRunnersResponse struct {
	Items      []Runner `json:"items"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type DrainRunnerRequest struct {
	Reason string `json:"reason,omitempty"`
}

type ListExecutionsResponse struct {
	Items      []Execution `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type CancelExecutionRequest struct {
	Reason string `json:"reason,omitempty"`
}

type ListAuditEventsResponse struct {
	Items      []AuditEvent `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

// Runner protocol request/response types.

type EnrollRequest struct {
	Token string `json:"token"`
}

type EnrollResponse struct {
	RunnerID    uuid.UUID `json:"runner_id"`
	AccessToken string    `json:"access_token"`
}

type HeartbeatRequest struct {
	Version       string   `json:"version"`
	DaemonVersion string   `json:"daemon_version"`
	Labels        Labels   `json:"labels"`
	Healthy       bool     `json:"healthy"`
	Assignments   []string `json:"assignments"`
}

type HeartbeatResponse struct {
	State RunnerState `json:"state"`
}

type PollAssignmentsRequest struct {
	AfterGeneration int64 `json:"after_generation"`
}

type PollAssignmentsResponse struct {
	Generation  int64        `json:"generation"`
	Assignments []Assignment `json:"assignments"`
}

type SubmitObservationsRequest struct {
	Observations []AssignmentObservation `json:"observations"`
}

type AssignmentObservation struct {
	AssignmentID uuid.UUID       `json:"assignment_id"`
	Generation   int64           `json:"generation"`
	State        AssignmentState `json:"state"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

type SubmitExecutionEventsRequest struct {
	Events []ExecutionEvent `json:"events"`
}

type SubmitLogChunksRequest struct {
	Chunks []LogChunk `json:"chunks"`
}

// Daemon API request/response types.

type DaemonAssignment struct {
	ID         uuid.UUID `json:"id"`
	ScheduleID uuid.UUID `json:"schedule_id"`
	Name       string    `json:"name"`
	Cron       string    `json:"cron"`
	Timezone   string    `json:"timezone,omitempty"`
	Timeout    string    `json:"timeout"`
	Command    []string  `json:"command"`
	Generation int64     `json:"generation"`
	Enabled    bool      `json:"enabled"`
}

type DaemonExecutionResult struct {
	ID           uuid.UUID      `json:"id"`
	AssignmentID uuid.UUID      `json:"assignment_id"`
	State        ExecutionState `json:"state"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty"`
	ExitCode     *int           `json:"exit_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type DaemonEventsResponse struct {
	Events   []DaemonEvent `json:"events"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

type DaemonEvent struct {
	ID           uuid.UUID      `json:"id"`
	ExecutionID  uuid.UUID      `json:"execution_id"`
	AssignmentID uuid.UUID      `json:"assignment_id"`
	State        ExecutionState `json:"state"`
	Message      string         `json:"message,omitempty"`
	EmittedAt    time.Time      `json:"emitted_at"`
	Sequence     int64          `json:"sequence"`
}
