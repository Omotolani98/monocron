package controller

import (
	"context"
	"time"

	"encore.dev/pubsub"
	"github.com/google/uuid"
)

// RunMessage is published for runs that should be executed by any runner.
// Multiple runner instances can subscribe with the same consumer group for load balancing.
type RunMessage struct {
	RunID       uuid.UUID `json:"run_id"`
	TaskID      uuid.UUID `json:"task_id"`
	TaskName    string    `json:"task_name"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Source      string    `json:"source"`
	Executor    jsonRaw   `json:"executor"`
}

// jsonRaw is a lightweight alias for raw JSON bytes in messages.
type jsonRaw []byte

//encore:topic name=run-dispatch
var RunDispatchTopic = pubsub.NewTopic[RunMessage]("run-dispatch", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

// publishRun broadcasts a run message to all runners (fan-out with group consumption).
func publishRun(ctx context.Context, msg RunMessage) error {
	_, err := RunDispatchTopic.Publish(ctx, msg)
	return err
}
