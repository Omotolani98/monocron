package cli

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

// CLIClient is the monocronctl HTTP client.
type CLIClient struct {
	baseURL string
	client  *http.Client
	apiKey  string
}

// NewCLIClient creates a CLI client from environment variables.
func NewCLIClient() *CLIClient {
	return &CLIClient{
		baseURL: strings.TrimSuffix(os.Getenv("MONOCRON_CONTROLLER_URL"), "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  os.Getenv("MONOCRON_API_KEY"),
	}
}

// Config holds persisted CLI configuration.
type Config struct {
	ControllerURL string `json:"controller_url"`
	APIKey        string `json:"api_key"`
}

// ConfigPath returns the default config path.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "monocronctl", "config.json")
}

// LoadConfig reads the CLI config from disk.
func LoadConfig() (Config, error) {
	var cfg Config
	b, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(b, &cfg)
	return cfg, err
}

// SaveConfig writes the CLI config to disk.
func SaveConfig(cfg Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o700); err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), b, 0o600)
}

func (c *CLIClient) get(ctx context.Context, path string, out any) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *CLIClient) post(ctx context.Context, path string, body, out any) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *CLIClient) put(ctx context.Context, path string, body, out any) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func (c *CLIClient) delete(ctx context.Context, path string) error {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, nil)
}

func (c *CLIClient) authorize(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("X-Actor", "monocronctl")
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

// Schedule API

func (c *CLIClient) CreateSchedule(ctx context.Context, req contracts.CreateScheduleRequest) (contracts.ScheduleSpec, error) {
	var resp contracts.ScheduleSpec
	err := c.post(ctx, "/api/v1/schedules", req, &resp)
	return resp, err
}

func (c *CLIClient) ListSchedules(ctx context.Context) (contracts.ListSchedulesResponse, error) {
	var resp contracts.ListSchedulesResponse
	err := c.get(ctx, "/api/v1/schedules", &resp)
	return resp, err
}

func (c *CLIClient) GetSchedule(ctx context.Context, id uuid.UUID) (contracts.ScheduleSpec, error) {
	var resp contracts.ScheduleSpec
	err := c.get(ctx, fmt.Sprintf("/api/v1/schedules/%s", id), &resp)
	return resp, err
}

func (c *CLIClient) UpdateSchedule(ctx context.Context, id uuid.UUID, req contracts.UpdateScheduleRequest) (contracts.ScheduleSpec, error) {
	var resp contracts.ScheduleSpec
	err := c.put(ctx, fmt.Sprintf("/api/v1/schedules/%s", id), req, &resp)
	return resp, err
}

func (c *CLIClient) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	return c.delete(ctx, fmt.Sprintf("/api/v1/schedules/%s", id))
}

// Runner API

func (c *CLIClient) CreateEnrollmentToken(ctx context.Context, req contracts.CreateEnrollmentTokenRequest) (contracts.CreateEnrollmentTokenResponse, error) {
	var resp contracts.CreateEnrollmentTokenResponse
	err := c.post(ctx, "/api/v1/enrollment-tokens", req, &resp)
	return resp, err
}

func (c *CLIClient) ListRunners(ctx context.Context) (contracts.ListRunnersResponse, error) {
	var resp contracts.ListRunnersResponse
	err := c.get(ctx, "/api/v1/runners", &resp)
	return resp, err
}

func (c *CLIClient) GetRunner(ctx context.Context, id uuid.UUID) (contracts.Runner, error) {
	var resp contracts.Runner
	err := c.get(ctx, fmt.Sprintf("/api/v1/runners/%s", id), &resp)
	return resp, err
}

func (c *CLIClient) DrainRunner(ctx context.Context, id uuid.UUID, req contracts.DrainRunnerRequest) error {
	return c.post(ctx, fmt.Sprintf("/api/v1/runners/%s/drain", id), req, nil)
}

func (c *CLIClient) DeleteRunner(ctx context.Context, id uuid.UUID) error {
	return c.delete(ctx, fmt.Sprintf("/api/v1/runners/%s", id))
}

// Execution API

func (c *CLIClient) ListExecutions(ctx context.Context) (contracts.ListExecutionsResponse, error) {
	var resp contracts.ListExecutionsResponse
	err := c.get(ctx, "/api/v1/executions", &resp)
	return resp, err
}

func (c *CLIClient) GetExecution(ctx context.Context, id uuid.UUID) (contracts.Execution, error) {
	var resp contracts.Execution
	err := c.get(ctx, fmt.Sprintf("/api/v1/executions/%s", id), &resp)
	return resp, err
}

func (c *CLIClient) GetExecutionLogs(ctx context.Context, id uuid.UUID) ([]contracts.LogChunk, error) {
	var resp []contracts.LogChunk
	err := c.get(ctx, fmt.Sprintf("/api/v1/executions/%s/logs", id), &resp)
	return resp, err
}

func (c *CLIClient) CancelExecution(ctx context.Context, id uuid.UUID, req contracts.CancelExecutionRequest) error {
	return c.post(ctx, fmt.Sprintf("/api/v1/executions/%s/cancel", id), req, nil)
}

// Audit API

func (c *CLIClient) ListAuditEvents(ctx context.Context) (contracts.ListAuditEventsResponse, error) {
	var resp contracts.ListAuditEventsResponse
	err := c.get(ctx, "/api/v1/audit-events", &resp)
	return resp, err
}
