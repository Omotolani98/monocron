package controllerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
	"github.com/google/uuid"
)

// Client talks to the monocron-controller over HTTPS.
type Client struct {
	baseURL    string
	httpClient *http.Client
	accessToken string
}

// New creates a controller client.
func New(baseURL, accessToken string) *Client {
	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 60 * time.Second},
		accessToken: accessToken,
	}
}

// SetAccessToken updates the bearer token used by the client.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// Enroll exchanges an enrollment token for runner credentials.
func (c *Client) Enroll(ctx context.Context, token string) (contracts.EnrollResponse, error) {
	var resp contracts.EnrollResponse
	req := contracts.EnrollRequest{Token: token}
	err := c.post(ctx, "/api/v1/runners/enroll", req, &resp)
	return resp, err
}

// Heartbeat sends a heartbeat.
func (c *Client) Heartbeat(ctx context.Context, req contracts.HeartbeatRequest) (contracts.HeartbeatResponse, error) {
	var resp contracts.HeartbeatResponse
	err := c.post(ctx, "/api/v1/runners/heartbeat", req, &resp)
	return resp, err
}

// PollAssignments long-polls for assignments.
func (c *Client) PollAssignments(ctx context.Context, after int64) (contracts.PollAssignmentsResponse, error) {
	var resp contracts.PollAssignmentsResponse
	u := fmt.Sprintf("/api/v1/runners/assignments?after=%d", after)
	err := c.get(ctx, u, &resp)
	return resp, err
}

// GetSchedule fetches a schedule by ID.
func (c *Client) GetSchedule(ctx context.Context, id uuid.UUID) (contracts.ScheduleSpec, error) {
	var resp contracts.ScheduleSpec
	err := c.get(ctx, fmt.Sprintf("/api/v1/schedules/%s", id), &resp)
	return resp, err
}

// SubmitObservations reports assignment states.
func (c *Client) SubmitObservations(ctx context.Context, observations []contracts.AssignmentObservation) error {
	req := contracts.SubmitObservationsRequest{Observations: observations}
	return c.post(ctx, "/api/v1/runners/observations", req, nil)
}

// SubmitExecutionEvents reports execution state changes.
func (c *Client) SubmitExecutionEvents(ctx context.Context, events []contracts.ExecutionEvent) error {
	req := contracts.SubmitExecutionEventsRequest{Events: events}
	return c.post(ctx, "/api/v1/runners/execution-events", req, nil)
}

// SubmitLogChunks reports log chunks.
func (c *Client) SubmitLogChunks(ctx context.Context, chunks []contracts.LogChunk) error {
	req := contracts.SubmitLogChunksRequest{Chunks: chunks}
	return c.post(ctx, "/api/v1/runners/log-chunks", req, nil)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	rel, err := url.Parse(path)
	if err != nil {
		return err
	}
	u := base.ResolveReference(rel)

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.authorize(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *Client) authorize(req *http.Request) {
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
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

// TokenStore persists runner credentials locally.
type TokenStore struct {
	Path string
}

// NewTokenStore creates a token store at the given path.
func NewTokenStore(path string) *TokenStore {
	return &TokenStore{Path: path}
}

// RunnerState is the persisted runner identity.
type RunnerState struct {
	RunnerID    uuid.UUID `json:"runner_id"`
	AccessToken string    `json:"access_token"`
}

// Load reads the runner state from disk.
func (t *TokenStore) Load() (RunnerState, error) {
	var s RunnerState
	b, err := os.ReadFile(t.Path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}

// Save writes the runner state to disk with restricted permissions.
func (t *TokenStore) Save(s RunnerState) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.Path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(t.Path, b, 0o600)
}

func init() {
	_ = uuid.New()
}
