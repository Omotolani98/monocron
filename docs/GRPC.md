# GRPC (Daemon ↔ Runner)

Files

- proto/daemon/v1/daemon.proto: Service + messages
- buf.yaml / buf.gen.yaml: Generation config for gRPC + Connect

Generate code (using buf)

- Install: <https://buf.build/docs/installation>
- Run from repo root:
  buf generate

Outputs (default paths)

- gen/daemon/v1/daemon.pb.go
- gen/daemon/v1/daemon_grpc.pb.go
- gen/daemon/v1/daemonv1connect/daemon.connect.go

Server (Daemon)

- gRPC:
  
  s := grpc.NewServer()
  daemonv1.RegisterDaemonServer(s, impl)

- Connect (HTTP/2 or HTTP/1.1):
  
  mux := http.NewServeMux()
  mux.Handle(daemonv1connect.NewDaemonHandler(impl))

Client (Runner)

- gRPC:
  
  conn, _:= grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
  cli := daemonv1.NewDaemonClient(conn)
  stream,_ := cli.Execute(ctx, &daemonv1.ExecuteRequest{...})
  for {
      ev, err := stream.Recv()
      if err != nil { break }
      // forward to pub/sub
  }

- Connect (HTTP):
  
  hc := &http.Client{Timeout: 10 _time.Second}
  cli := daemonv1connect.NewDaemonClient(hc, baseURL)
  stream, _ := cli.Execute(ctx, connect.NewRequest(&daemonv1.ExecuteRequest{...}))
  for stream.Receive() { ev := stream.Msg(); /_ forward */ }

Notes

- After generating code, refactor runner/daemon to use generated stubs and remove the JSON gRPC shim.
- Pub/Sub wiring remains unchanged.
