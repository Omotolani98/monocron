package daemonclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

// Client talks to monocrond over a Unix socket.
type Client struct {
	socketPath string
	httpClient *http.Client
}

// New creates a daemon client.
func New(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

// Health checks daemon health.
func (c *Client) Health(ctx context.Context) error {
	return c.get(ctx, "/v1/health", nil)
}

// ApplyAssignment sends an assignment to the daemon.
func (c *Client) ApplyAssignment(ctx context.Context, a contracts.DaemonAssignment) error {
	return c.put(ctx, fmt.Sprintf("/v1/assignments/%s", a.ID), a, nil)
}

// DeleteAssignment removes an assignment from the daemon.
func (c *Client) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	return c.delete(ctx, fmt.Sprintf("/v1/assignments/%s", id))
}

// ListAssignments returns assignments from the daemon.
func (c *Client) ListAssignments(ctx context.Context) ([]contracts.DaemonAssignment, error) {
	var out []contracts.DaemonAssignment
	err := c.get(ctx, "/v1/assignments", &out)
	return out, err
}

// ListExecutions returns executions for an assignment.
func (c *Client) ListExecutions(ctx context.Context, assignmentID uuid.UUID) ([]contracts.DaemonExecutionResult, error) {
	var out []contracts.DaemonExecutionResult
	err := c.get(ctx, fmt.Sprintf("/v1/assignments/%s/executions", assignmentID), &out)
	return out, err
}

// GetExecution returns a single execution from the daemon.
func (c *Client) GetExecution(ctx context.Context, id uuid.UUID) (contracts.DaemonExecutionResult, error) {
	var out contracts.DaemonExecutionResult
	err := c.get(ctx, fmt.Sprintf("/v1/executions/%s", id), &out)
	return out, err
}

// GetExecutionLogs returns log chunks for an execution.
func (c *Client) GetExecutionLogs(ctx context.Context, id uuid.UUID) ([]contracts.LogChunk, error) {
	var out []contracts.LogChunk
	err := c.get(ctx, fmt.Sprintf("/v1/executions/%s/logs", id), &out)
	return out, err
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	u := "http://unix" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *Client) put(ctx context.Context, path string, body, out any) error {
	u := "http://unix" + path
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	u := "http://unix" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, nil)
}

func decodeResponse(resp *http.Response, out any) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
