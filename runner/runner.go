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
