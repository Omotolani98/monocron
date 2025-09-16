package monocrond

import (
	"context"

	"connectrpc.com/connect"
	daemonv1 "encore.app/gen/daemon/v1"
	"github.com/charmbracelet/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MonocrondServer struct{}

func (s *MonocrondServer) Execute(
	ctx context.Context,
	req *connect.Request[daemonv1.ExecuteRequest],
	resp *connect.ServerStream[daemonv1.ExecuteResponse],
) error {
	log.Infof("Request Headers: %s", req.Header())
	res := connect.NewResponse(&daemonv1.ExecuteResponse{
		Event: &daemonv1.ExecuteResponse_Status{
			Status: &daemonv1.StatusEvent{
				At:     timestamppb.Now(),
				Status: "RUNNING",
			},
		},
	})

	log.Infof("%v", res)

	return nil
}
