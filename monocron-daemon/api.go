package monocrond

import (
    "time"
)

// ExecuteRequest describes a job to run.
type ExecuteRequest struct {
    RunID      string   `json:"run_id"`
    TaskName   string   `json:"task_name"`
    Command    []string `json:"command"`   // command[0] is the binary, rest are args
    Env        []string `json:"env"`       // KEY=VALUE
    WorkingDir string   `json:"working_dir"`
}

// ExecuteEvent is streamed from daemon to runner.
type ExecuteEvent struct {
    Type   string       `json:"type"` // "log" | "status"
    Log    *LogEvent    `json:"log,omitempty"`
    Status *StatusEvent `json:"status,omitempty"`
}

type LogEvent struct {
    At   time.Time `json:"at"`
    Line string    `json:"line"`
    Std  string    `json:"std"` // "out" | "err"
}

type StatusEvent struct {
    At     time.Time `json:"at"`
    Status string    `json:"status"` // RUNNING|COMPLETED|FAILED
    Error  string    `json:"error,omitempty"`
    Code   int       `json:"code,omitempty"`
}

