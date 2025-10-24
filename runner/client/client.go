package client

import (
	"context"
	"net"
	"net/http"
)

type UnixConn struct {
	Conn net.Conn
}

// const tempFile = "/tmp/monocron.sock"
const socketFile = "/run/monocron/monocron.sock"

func InitMuxClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketFile)
		},
	}

	return &http.Client{
		Transport: transport,
		// Timeout:   5 * time.Second,
	}
}
