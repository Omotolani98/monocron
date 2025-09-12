//encore:service
package runner

import (
	"context"

	"encore.app/controller"
	"github.com/charmbracelet/log"
)

// HandleRun subscribes to run dispatches. Multiple runner instances can use the
// same consumer group for load-balanced consumption in heterogeneous environments.
//
//encore:subscribe topic=controller.RunDispatchTopic
func HandleRun(ctx context.Context, msg *controller.RunMessage) error {
	// TODO: Integrate monocrond + sandbox exec here.
	// For now, just acknowledge by returning nil.
	log.Info(msg)

	return nil
}
