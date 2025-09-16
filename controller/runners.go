package controller

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
)

type RegisterRunnerRequest struct {
    Kind string `json:"kind"` // vm | docker | bare-metal
}

type RegisterRunnerResponse struct {
    RunnerID uuid.UUID `json:"runner_id"`
}

//encore:api public method=POST path=/runners/register
func RegisterRunner(ctx context.Context, req *RegisterRunnerRequest) (*RegisterRunnerResponse, error) {
    if req == nil || req.Kind == "" {
        return nil, errors.New("kind is required")
    }
    var id uuid.UUID
    // Insert and return id
    row := pgxdb.QueryRow(ctx, `
        INSERT INTO runners(kind, status, last_seen)
        VALUES ($1, 'live', NOW())
        RETURNING id
    `, req.Kind)
    if err := row.Scan(&id); err != nil {
        return nil, err
    }
    return &RegisterRunnerResponse{RunnerID: id}, nil
}

type HeartbeatRequest struct {
    RunnerID uuid.UUID `json:"runner_id"`
}

//encore:api public method=POST path=/runners/heartbeat
func Heartbeat(ctx context.Context, req *HeartbeatRequest) error {
    if req == nil || req.RunnerID == uuid.Nil {
        return errors.New("runner_id required")
    }
    _, err := pgxdb.Exec(ctx, `
        UPDATE runners
        SET last_seen = NOW(), status = 'live'
        WHERE id = $1
    `, req.RunnerID)
    return err
}

type Runner struct {
    ID       uuid.UUID `json:"runner_id"`
    Kind     string    `json:"kind"`
    Status   string    `json:"status"`
    LastSeen time.Time `json:"last_seen"`
    Created  time.Time `json:"created_at"`
}

type ListRunnersResponse struct {
    Runners []Runner `json:"runners"`
}

//encore:api public method=GET path=/runners
func ListRunners(ctx context.Context) (*ListRunnersResponse, error) {
    rows, err := pgxdb.Query(ctx, `
        SELECT id, kind, status, last_seen, created_at
        FROM runners
        ORDER BY last_seen DESC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    out := make([]Runner, 0, 16)
    now := time.Now().UTC()
    for rows.Next() {
        var r Runner
        if err := rows.Scan(&r.ID, &r.Kind, &r.Status, &r.LastSeen, &r.Created); err != nil {
            return nil, err
        }
        // Compute liveness (dead if >60s since last_seen)
        if now.Sub(r.LastSeen) > 60*time.Second {
            r.Status = "dead"
        } else {
            r.Status = "live"
        }
        out = append(out, r)
    }
    return &ListRunnersResponse{Runners: out}, nil
}

type RunnerActionRequest struct {
    RunnerID uuid.UUID `json:"runner_id"`
}

//encore:api public method=POST path=/runners/stop
func StopRunner(ctx context.Context, req *RunnerActionRequest) error {
    if req == nil || req.RunnerID == uuid.Nil {
        return errors.New("runner_id required")
    }
    return publishRunnerControl(ctx, RunnerControlMessage{RunnerID: req.RunnerID, Action: "STOP"})
}

//encore:api public method=POST path=/runners/kill
func KillRunner(ctx context.Context, req *RunnerActionRequest) error {
    if req == nil || req.RunnerID == uuid.Nil {
        return errors.New("runner_id required")
    }
    return publishRunnerControl(ctx, RunnerControlMessage{RunnerID: req.RunnerID, Action: "KILL"})
}

// --- Join API ---

type JoinRunnerRequest struct {
    // If provided, we refresh the existing runner; otherwise we create a new one.
    RunnerID *uuid.UUID `json:"runner_id,omitempty"`
    Kind     string     `json:"kind"`
}

type JoinRunnerResponse struct {
    RunnerID            uuid.UUID `json:"runner_id"`
    DispatchTopic       string    `json:"dispatch_topic"`
    ControlTopic        string    `json:"control_topic"`
    HeartbeatIntervalSec int      `json:"heartbeat_interval_sec"`
}

//encore:api public method=POST path=/runners/join
func JoinRunner(ctx context.Context, req *JoinRunnerRequest) (*JoinRunnerResponse, error) {
    if req == nil || req.Kind == "" {
        return nil, errors.New("kind is required")
    }
    var id uuid.UUID
    if req.RunnerID != nil && *req.RunnerID != uuid.Nil {
        id = *req.RunnerID
        _, err := pgxdb.Exec(ctx, `
            UPDATE runners SET last_seen = NOW(), status = 'live', kind = $2 WHERE id = $1
        `, id, req.Kind)
        if err != nil {
            return nil, err
        }
    } else {
        row := pgxdb.QueryRow(ctx, `
            INSERT INTO runners(kind, status, last_seen)
            VALUES ($1, 'live', NOW())
            RETURNING id
        `, req.Kind)
        if err := row.Scan(&id); err != nil {
            return nil, err
        }
    }
    return &JoinRunnerResponse{
        RunnerID:            id,
        DispatchTopic:       "run-dispatch",
        ControlTopic:        "runner-control",
        HeartbeatIntervalSec: 30,
    }, nil
}
