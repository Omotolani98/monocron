//encore:service
package runner

import (
	"context"

	"encore.app/controller"
	"encore.dev/pubsub"
	"github.com/charmbracelet/log"
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
	// TODO: Integrate monocrond + sandbox exec here.
	// For now, just acknowledge by returning nil.
	log.Info(msg)

	return nil
}
