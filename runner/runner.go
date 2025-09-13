//encore:service
package runner

import (
    "context"
    "fmt"
    "time"

    "encore.app/controller"
    "encore.dev/pubsub"
    "github.com/charmbracelet/log"
    "github.com/google/uuid"
)

var _ = pubsub.NewSubscription(
	controller.RunDispatchTopic,
	"run-dispatch",
	pubsub.SubscriptionConfig[controller.RunMessage]{
		Handler: HandleRun,
	},
)

// HandleRun subscribes to run dispatches. Multiple runner instances can use the
// same consumer group for load-balanced consumption in heterogeneous environments.
func HandleRun(ctx context.Context, msg controller.RunMessage) error {
    // Mark RUNNING
    _ = publishStatus(ctx, msg.RunID, "RUNNING", "")

    // TODO: call local monocrond and stream its output.
    // Simulate execution with a few log lines.
    for i := 1; i <= 3; i++ {
        _ = publishLog(ctx, msg.RunID, fmt.Sprintf("processing step %d for %s", i, msg.TaskName))
        time.Sleep(100 * time.Millisecond)
    }

    // Mark COMPLETED (or FAILED on error)
    _ = publishStatus(ctx, msg.RunID, "COMPLETED", "")
    log.Info("run completed", "run_id", msg.RunID)
    return nil
}

func publishLog(ctx context.Context, runID uuid.UUID, line string) error {
    _, err := controller.RunLogsTopic.Publish(ctx, controller.LogMessage{RunID: runID, LoggedAt: time.Now().UTC(), Line: line})
    return err
}

func publishStatus(ctx context.Context, runID uuid.UUID, status, errStr string) error {
    _, err := controller.RunStatusTopic.Publish(ctx, controller.StatusMessage{RunID: runID, Status: status, Error: errStr, OccurredAt: time.Now().UTC()})
    return err
}

// --- RPC endpoint for daemon push model ---

// DaemonEvent represents an event sent by the local monocrond to the runner.
type DaemonEvent struct {
    Type   string           `json:"type"` // "log" | "status"
    RunID  string           `json:"run_id"`
    Log    *DaemonLog       `json:"log,omitempty"`
    Status *DaemonStatus    `json:"status,omitempty"`
}

type DaemonLog struct {
    Line string    `json:"line"`
    Std  string    `json:"std"`
    At   time.Time `json:"at"`
}

type DaemonStatus struct {
    Status string    `json:"status"`
    Error  string    `json:"error,omitempty"`
    Code   int       `json:"code,omitempty"`
    At     time.Time `json:"at"`
}

//encore:api public method=POST path=/daemon/event
func ReceiveDaemonEvent(ctx context.Context, ev *DaemonEvent) error {
    if ev == nil {
        return nil
    }
    rid, err := uuid.Parse(ev.RunID)
    if err != nil {
        return fmt.Errorf("invalid run_id: %w", err)
    }
    switch ev.Type {
    case "log":
        if ev.Log != nil {
            return publishLog(ctx, rid, ev.Log.Line)
        }
    case "status":
        if ev.Status != nil {
            return publishStatus(ctx, rid, ev.Status.Status, ev.Status.Error)
        }
    }
    return nil
}
