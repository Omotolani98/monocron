package controller

import (
	"context"

	"encore.app/controller/db"
	"encore.dev/pubsub"
)

var _ = pubsub.NewSubscription(
	RunLogsTopic,
	"run-logs",
	pubsub.SubscriptionConfig[LogMessage]{
		Handler: IngestRunLog,
	},
)

var _ = pubsub.NewSubscription(
	RunStatusTopic,
	"run-status",
	pubsub.SubscriptionConfig[StatusMessage]{
		Handler: IngestRunStatus,
	},
)

// Persist runner log lines.
func IngestRunLog(ctx context.Context, msg LogMessage) error {
	_, err := q.InsertLog(ctx, db.InsertLogParams{RunID: msg.RunID, LogLine: msg.Line})
	return err
}

// Update run status based on runner events.
func IngestRunStatus(ctx context.Context, msg StatusMessage) error {
	switch msg.Status {
	case "RUNNING":
		return q.MarkAsRunning(ctx, msg.RunID)
	case "COMPLETED":
		return q.MarkRunCompleted(ctx, msg.RunID)
	case "FAILED":
		return q.MarkRunFailed(ctx, msg.RunID)
	default:
		return nil
	}
}
