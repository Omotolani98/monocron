package monocrond

import "time"

// ExecuteRequest describes a job to run via daemon RPC.
type ExecuteRequest struct {
    RunID      string   `json:"run_id"`
    TaskName   string   `json:"task_name"`
    Command    []string `json:"command"`
    Env        []string `json:"env,omitempty"`
    WorkingDir string   `json:"working_dir,omitempty"`
}

// ExecuteEvent is streamed back from the daemon.
type ExecuteEvent struct {
    Type   string       `json:"type"` // "log" | "status"
    Log    *LogEvent    `json:"log,omitempty"`
    Status *StatusEvent `json:"status,omitempty"`
}

type LogEvent struct {
    At   time.Time `json:"at"`
    Line string    `json:"line"`
    Std  string    `json:"std"` // out|err
}

type StatusEvent struct {
    At     time.Time `json:"at"`
    Status string    `json:"status"` // RUNNING|COMPLETED|FAILED
    Error  string    `json:"error,omitempty"`
    Code   int       `json:"code,omitempty"`
}

