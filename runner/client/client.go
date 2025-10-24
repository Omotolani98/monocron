package client

import (
	"context"
	"net"
	"net/http"
)

type UnixConn struct {
	Conn net.Conn
}

const tempFile = "/tmp/monocron.sock"

// func InitUnixClient() (*net.Conn, error) {
// 	conn, err := net.Dial("unix", tempFile)
// 	if err != nil {
// 		return nil, fmt.Errorf("Cannot connect to monocron.sock |> %v", err)
// 	}
// 	defer conn.Close()

// 	client := &http.Client{
// 		Transport: conn,
// 		Timeout:   5 * time.Second,
// 	}

// 	return &conn, nil
// }

func InitMuxClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", tempFile)
		},
	}

	return &http.Client{
		Transport: transport,
		// Timeout:   5 * time.Second,
	}
}
