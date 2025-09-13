package monocrond

import (
    "bufio"
    "context"
    "errors"
    "io"
    "net"
    "net/http"
    "os"
    "os/exec"
    "bytes"
    "encoding/json"
    "strings"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/encoding"
)

const (
    serviceName  = "monocrond.Daemon"
    methodExecute = "/" + serviceName + "/Execute"
)

// Register server with a custom JSON codec.
func init() { encoding.RegisterCodec(jsonCodec{}) }

var (
    httpClient      = &http.Client{Timeout: 5 * time.Second}
    runnerEventURL  string
)

func init() {
    // If RUNNER_EVENT_URL is set, use it directly, else build from RUNNER_BASE_URL
    if u := os.Getenv("RUNNER_EVENT_URL"); u != "" {
        runnerEventURL = u
    } else if b := strings.TrimRight(os.Getenv("RUNNER_BASE_URL"), "/"); b != "" {
        runnerEventURL = b + "/daemon/event"
    }
}

// Daemon implements the execution service.
type Daemon struct{}

// Serve starts the gRPC server on the given address, e.g. ":50051".
func Serve(ctx context.Context, addr string) error {
    lis, err := net.Listen("tcp", addr)
    if err != nil { return err }
    s := grpc.NewServer()
    type daemonIface interface{}
    s.RegisterService(&grpc.ServiceDesc{
        ServiceName: serviceName,
        HandlerType: (*daemonIface)(nil),
        Streams: []grpc.StreamDesc{{
            StreamName:    "Execute",
            Handler:       executeHandler,
            ServerStreams: true,
        }},
    }, &Daemon{})

    go func() {
        <-ctx.Done()
        s.GracefulStop()
        _ = lis.Close()
    }()
    return s.Serve(lis)
}

// executeHandler is the low-level gRPC handler for server-streaming Execute.
func executeHandler(srv interface{}, stream grpc.ServerStream) error {
    var req ExecuteRequest
    if err := stream.RecvMsg(&req); err != nil { return err }
    d := srv.(*Daemon)
    return d.execute(stream.Context(), &req, stream)
}

// execute runs the command and streams logs & status.
func (d *Daemon) execute(ctx context.Context, req *ExecuteRequest, stream grpc.ServerStream) error {
    if len(req.Command) == 0 { return errors.New("empty command") }
    cmd := exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
    if req.WorkingDir != "" { cmd.Dir = req.WorkingDir }
    if len(req.Env) > 0 { cmd.Env = append(os.Environ(), req.Env...) }

    stdout, err := cmd.StdoutPipe(); if err != nil { return err }
    stderr, err := cmd.StderrPipe(); if err != nil { return err }

    if err := cmd.Start(); err != nil {
        _ = sendStatus(stream, req.RunID, "FAILED", err, -1)
        return err
    }
    _ = sendStatus(stream, req.RunID, "RUNNING", nil, 0)

    done := make(chan struct{}, 2)
    go forward(req.RunID, "out", stdout, stream, done)
    go forward(req.RunID, "err", stderr, stream, done)
    <-done; <-done // wait both

    err = cmd.Wait()
    if err != nil {
        _ = sendStatus(stream, req.RunID, "FAILED", err, exitCode(err))
        return nil
    }
    _ = sendStatus(stream, req.RunID, "COMPLETED", nil, 0)
    return nil
}

func forward(runID, std string, r io.Reader, stream grpc.ServerStream, done chan<- struct{}) {
    defer func(){ done<-struct{}{} }()
    s := bufio.NewScanner(r)
    for s.Scan() {
        ev := &ExecuteEvent{Type: "log", Log: &LogEvent{At: time.Now().UTC(), Line: s.Text(), Std: std}}
        _ = stream.SendMsg(ev)
        postEvent(runID, ev)
    }
}

func sendStatus(stream grpc.ServerStream, runID, st string, err error, code int) error {
    ev := &ExecuteEvent{Type: "status", Status: &StatusEvent{At: time.Now().UTC(), Status: st, Code: code}}
    if err != nil { ev.Status.Error = err.Error() }
    postEvent(runID, ev)
    return stream.SendMsg(ev)
}

// exitCode tries to extract an exit code from exec error.
func exitCode(err error) int {
    if err == nil { return 0 }
    var ee *exec.ExitError
    if errors.As(err, &ee) {
        if ee.ProcessState != nil { return ee.ProcessState.ExitCode() }
    }
    return -1
}

// postEvent optionally POSTs the event to the runner's HTTP endpoint when configured.
func postEvent(runID string, ev *ExecuteEvent) {
    if runnerEventURL == "" || ev == nil { return }
    payload := struct{
        RunID string `json:"run_id"`
        Type  string `json:"type"`
        Log   *LogEvent `json:"log,omitempty"`
        Status *StatusEvent `json:"status,omitempty"`
    }{RunID: runID, Type: ev.Type, Log: ev.Log, Status: ev.Status}

    b, err := json.Marshal(payload)
    if err != nil { return }
    req, err := http.NewRequest(http.MethodPost, runnerEventURL, bytes.NewReader(b))
    if err != nil { return }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "monocrond/1.0")
    // Fire-and-forget with short timeout (already set on client)
    _, _ = httpClient.Do(req)
}
