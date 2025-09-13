//encore:service
package runner

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    monocrond "encore.app/monocrond"
    "encore.app/controller"
    "encore.dev/pubsub"
    "github.com/charmbracelet/log"
    "github.com/google/uuid"
    "google.golang.org/grpc"
)

const grpcMethodExecute = "/monocrond.Daemon/Execute"

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
    // Connect to local monocrond (JSON gRPC stream)
    addr := getenv("MONOCROND_ADDR", ":50051")
    cc, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.ForceCodec(monocrond.JsonCodecForClient())))
    if err != nil {
        _ = publishStatus(ctx, msg.RunID, "FAILED", fmt.Sprintf("dial monocrond: %v", err))
        return nil
    }
    defer cc.Close()

    // Start Execute server stream
    desc := &grpc.StreamDesc{ServerStreams: true}
    stream, err := cc.NewStream(ctx, desc, grpcMethodExecute)
    if err != nil {
        _ = publishStatus(ctx, msg.RunID, "FAILED", fmt.Sprintf("start stream: %v", err))
        return nil
    }

    // Build ExecuteRequest from executor payload
    cmd := extractCommand([]byte(msg.Executor))
    req := &monocrond.ExecuteRequest{RunID: msg.RunID.String(), TaskName: msg.TaskName, Command: cmd}
    if err := stream.SendMsg(req); err != nil {
        _ = publishStatus(ctx, msg.RunID, "FAILED", fmt.Sprintf("send req: %v", err))
        _ = stream.CloseSend()
        return nil
    }
    _ = stream.CloseSend()

    // Forward daemon events to controller topics
    for {
        var ev monocrond.ExecuteEvent
        if err := stream.RecvMsg(&ev); err != nil {
            break
        }
        switch ev.Type {
        case "status":
            if ev.Status != nil {
                _ = publishStatus(ctx, msg.RunID, ev.Status.Status, ev.Status.Error)
            }
        case "log":
            if ev.Log != nil {
                _ = publishLog(ctx, msg.RunID, ev.Log.Line)
            }
        }
    }
    log.Info("run finished", "run_id", msg.RunID)
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

func getenv(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }

// extractCommand expects executor JSON like {"command":["/bin/echo","hello"]}
func extractCommand(raw []byte) []string {
    var v struct{ Command []string `json:"command"` }
    _ = json.Unmarshal(raw, &v)
    if len(v.Command) == 0 { return []string{"/bin/echo", "no-executor-command"} }
    return v.Command
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
