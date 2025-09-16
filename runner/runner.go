//encore:service
package runner

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "connectrpc.com/connect"
    "encore.app/controller"
    daemonv1 "encore.app/gen/daemon/v1"
    daemonv1connect "encore.app/gen/daemon/v1/daemonv1connect"
    "encore.dev/pubsub"
    "github.com/charmbracelet/log"
    "github.com/google/uuid"
)

const grpcMethodExecute = "/monocrond.Daemon/Execute"

var _ = pubsub.NewSubscription(
    controller.RunDispatchTopic,
    "run-dispatch",
    pubsub.SubscriptionConfig[controller.RunMessage]{
        Handler: HandleRun,
    },
)

// Listen for controller control messages for this runner.
var _ = pubsub.NewSubscription(
    controller.RunnerControlTopic,
    "runner-control",
    pubsub.SubscriptionConfig[controller.RunnerControlMessage]{
        Handler: HandleControl,
    },
)

func HandleRun(ctx context.Context, msg controller.RunMessage) error {
	// Connect to local monocrond (Connect over h2c/HTTP)
	baseURL := getenv("MONOCROND_BASE_URL", "http://localhost:50051")
	httpClient := &http.Client{}
	cli := daemonv1connect.NewDaemonServiceClient(httpClient, baseURL)
	stream, err := cli.Execute(ctx, connect.NewRequest(&daemonv1.ExecuteRequest{RunId: msg.RunID.String(), TaskName: msg.TaskName, Command: extractCommand([]byte(msg.Executor))}))
	if err != nil {
		_ = publishStatus(ctx, msg.RunID, "FAILED", fmt.Sprintf("start stream: %v", err))
		return nil
	}

    // Forward daemon events to controller topics
    for stream.Receive() {
        ev := stream.Msg()
        if st := ev.GetStatus(); st != nil {
            _ = publishStatus(ctx, msg.RunID, st.Status, st.Error)
        }
        if lg := ev.GetLog(); lg != nil {
            _ = publishLog(ctx, msg.RunID, lg.Line)
        }
    }
	if err := stream.Err(); err != nil {
		_ = publishStatus(ctx, msg.RunID, "FAILED", fmt.Sprintf("stream err: %v", err))
	}
	log.Info("run finished", "run_id", msg.RunID)
	return nil
}

// HandleControl reacts to STOP/KILL messages; in this skeleton we just log.
func HandleControl(ctx context.Context, msg controller.RunnerControlMessage) error {
    // Only act if control is directed at this runner instance
    if myRunnerID != uuid.Nil && msg.RunnerID != myRunnerID {
        return nil
    }
    log.Info("runner control received", "action", msg.Action, "runner_id", msg.RunnerID)
    return nil
}

// --- Join + Heartbeat lifecycle ---

var myRunnerID uuid.UUID

func init() {
    go func() {
        // Give service a brief moment to start up fully
        time.Sleep(200 * time.Millisecond)
        joinAndHeartbeat()
    }()
}

func joinAndHeartbeat() {
    // Determine runner kind
    kind := getenv("RUNNER_KIND", "docker")
    // Optional existing id
    var existing *uuid.UUID
    if v := os.Getenv("RUNNER_ID"); v != "" {
        if id, err := uuid.Parse(v); err == nil {
            existing = &id
        }
    }
    // Join controller
    resp, err := controller.JoinRunner(context.Background(), &controller.JoinRunnerRequest{RunnerID: existing, Kind: kind})
    if err != nil {
        log.Error("join failed", "err", err)
        return
    }
    myRunnerID = resp.RunnerID
    _ = os.Setenv("RUNNER_ID", myRunnerID.String())
    hbSec := resp.HeartbeatIntervalSec
    if hbSec <= 0 { hbSec = 30 }
    log.Info("joined controller", "runner_id", myRunnerID, "hb_sec", hbSec)

    // Heartbeat loop
    ticker := time.NewTicker(time.Duration(hbSec) * time.Second)
    defer ticker.Stop()
    for {
        <-ticker.C
        _ = controller.Heartbeat(context.Background(), &controller.HeartbeatRequest{RunnerID: myRunnerID})
    }
}

func publishLog(ctx context.Context, runID uuid.UUID, line string) error {
	_, err := controller.RunLogsTopic.Publish(ctx, controller.LogMessage{RunID: runID, LoggedAt: time.Now().UTC(), Line: line})
	return err
}

func publishStatus(ctx context.Context, runID uuid.UUID, status, errStr string) error {
	_, err := controller.RunStatusTopic.Publish(ctx, controller.StatusMessage{RunID: runID, Status: status, Error: errStr, OccurredAt: time.Now().UTC()})
	return err
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// extractCommand expects executor JSON like {"command":["/bin/echo","hello"]}
func extractCommand(raw []byte) []string {
	var v struct {
		Command []string `json:"command"`
	}
	_ = json.Unmarshal(raw, &v)
	if len(v.Command) == 0 {
		return []string{"/bin/echo", "no-executor-command"}
	}
	return v.Command
}

// --- RPC endpoint for daemon push model ---

// DaemonEvent represents an event sent by the local monocrond to the runner.
type DaemonEvent struct {
	Type   string        `json:"type"` // "log" | "status"
	RunID  string        `json:"run_id"`
	Log    *DaemonLog    `json:"log,omitempty"`
	Status *DaemonStatus `json:"status,omitempty"`
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
