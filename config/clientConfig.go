package config

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Omotolani98/monocron-runner/pkg/gen"
	"github.com/charmbracelet/log"
)

type Client struct {
	C gen.SchedulerClient
}

func ClientConfig() *Client {
	var conn *grpc.ClientConn
	conn, err := grpc.NewClient(":9000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Close()

	c := gen.NewSchedulerClient(conn)
	return &Client{
		C: c,
	}
}
