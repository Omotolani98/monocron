// Package config provides Viper-based configuration for the monocron-runner.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Defaults
const (
	DefaultControllerURL = "http://127.0.0.1:8080"
	DefaultDaemonSocket  = "/run/monocron/monocrond.sock"
	DefaultRunnerType    = "bare_metal"
	DefaultLogLevel      = "info"
)

// RunnerConfig holds runner agent configuration.
type RunnerConfig struct {
	ControllerURL     string            `json:"controller_url" mapstructure:"controller_url"`
	DaemonSocket      string            `json:"daemon_socket" mapstructure:"daemon_socket"`
	StatePath         string            `json:"state_path" mapstructure:"state_path"`
	Type              string            `json:"type" mapstructure:"type"`
	Labels            map[string]string `json:"labels" mapstructure:"labels"`
	PollInterval      time.Duration     `json:"poll_interval" mapstructure:"poll_interval"`
	HeartbeatInterval time.Duration     `json:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	LogLevel          string            `json:"log_level" mapstructure:"log_level"`
}

// MarshalJSON serializes durations as human-readable strings.
func (c RunnerConfig) MarshalJSON() ([]byte, error) {
	type alias RunnerConfig
	return json.Marshal(&struct {
		*alias
		PollInterval      string `json:"poll_interval"`
		HeartbeatInterval string `json:"heartbeat_interval"`
	}{
		alias:             (*alias)(&c),
		PollInterval:      c.PollInterval.String(),
		HeartbeatInterval: c.HeartbeatInterval.String(),
	})
}

// UnmarshalJSON parses durations from strings or numbers.
func (c *RunnerConfig) UnmarshalJSON(b []byte) error {
	type alias RunnerConfig
	aux := &struct {
		*alias
		PollInterval      string `json:"poll_interval"`
		HeartbeatInterval string `json:"heartbeat_interval"`
	}{alias: (*alias)(c)}
	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	if aux.PollInterval != "" {
		d, err := time.ParseDuration(aux.PollInterval)
		if err != nil {
			return fmt.Errorf("poll_interval: %w", err)
		}
		c.PollInterval = d
	}
	if aux.HeartbeatInterval != "" {
		d, err := time.ParseDuration(aux.HeartbeatInterval)
		if err != nil {
			return fmt.Errorf("heartbeat_interval: %w", err)
		}
		c.HeartbeatInterval = d
	}
	return nil
}

// DefaultConfig returns the built-in runner defaults.
func DefaultConfig() RunnerConfig {
	return RunnerConfig{
		ControllerURL:     DefaultControllerURL,
		DaemonSocket:      DefaultDaemonSocket,
		Type:              DefaultRunnerType,
		Labels:            map[string]string{},
		PollInterval:      10 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		LogLevel:          DefaultLogLevel,
	}
}

// DefaultConfigPath returns the platform-specific runner config file path.
func DefaultConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "monocron", "runner.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "monocron", "runner.json")
}

// DefaultStatePath returns the platform-specific runner state file path.
func DefaultStatePath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "monocron", "runner.state")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "monocron", "runner.state")
}

// ReadConfig reads the runner config file from disk without creating defaults.
func ReadConfig(path string) (RunnerConfig, error) {
	var cfg RunnerConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// EnsureConfig creates the runner config directory and a default config file
// when path does not exist. It never overwrites an existing file.
func EnsureConfig(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return writeConfig(path, DefaultConfig())
}

// writeConfig writes cfg to path atomically with 0600 permissions.
func writeConfig(path string, cfg RunnerConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads the runner config from path, creating a default file if needed,
// and applies environment variable overrides using Viper.
func Load(path string) (RunnerConfig, error) {
	if err := EnsureConfig(path); err != nil {
		return RunnerConfig{}, fmt.Errorf("ensure config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	if err := v.ReadInConfig(); err != nil {
		return RunnerConfig{}, fmt.Errorf("read config: %w", err)
	}

	v.SetEnvPrefix("MONOCRON")
	v.AutomaticEnv()

	// Bind runner-specific environment variables to their config keys so the
	// documented names (e.g. MONOCRON_RUNNER_STATE_PATH) take precedence.
	_ = v.BindEnv("state_path", "MONOCRON_RUNNER_STATE_PATH")
	_ = v.BindEnv("type", "MONOCRON_RUNNER_TYPE")
	_ = v.BindEnv("labels", "MONOCRON_RUNNER_LABELS")
	_ = v.BindEnv("poll_interval", "MONOCRON_POLL_INTERVAL", "MONOCRON_RUNNER_POLL_INTERVAL")
	_ = v.BindEnv("heartbeat_interval", "MONOCRON_HEARTBEAT_INTERVAL", "MONOCRON_RUNNER_HEARTBEAT_INTERVAL")

	v.SetDefault("controller_url", DefaultControllerURL)
	v.SetDefault("daemon_socket", DefaultDaemonSocket)
	v.SetDefault("type", DefaultRunnerType)
	v.SetDefault("poll_interval", 10*time.Second)
	v.SetDefault("heartbeat_interval", 10*time.Second)
	v.SetDefault("log_level", DefaultLogLevel)

	cfg := populatedConfig(v)
	if cfg.StatePath == "" {
		cfg.StatePath = DefaultStatePath()
	}
	return cfg, nil
}

// populatedConfig reads fields from a Viper instance.
func populatedConfig(v *viper.Viper) RunnerConfig {
	labels := loadLabels(v)
	if labels == nil {
		labels = map[string]string{}
	}
	return RunnerConfig{
		ControllerURL:     v.GetString("controller_url"),
		DaemonSocket:      v.GetString("daemon_socket"),
		StatePath:         v.GetString("state_path"),
		Type:              v.GetString("type"),
		Labels:            labels,
		PollInterval:      v.GetDuration("poll_interval"),
		HeartbeatInterval: v.GetDuration("heartbeat_interval"),
		LogLevel:          v.GetString("log_level"),
	}
}

// loadLabels reads labels from config or from a comma-separated env string.
func loadLabels(v *viper.Viper) map[string]string {
	if v.IsSet("labels") {
		if s := v.GetString("labels"); s != "" {
			parsed, err := parseLabels(s)
			if err == nil {
				return parsed
			}
		}
	}
	return v.GetStringMapString("labels")
}

func parseLabels(v string) (map[string]string, error) {
	labels := map[string]string{}
	if v == "" {
		return labels, nil
	}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label %q, expected key=value", part)
		}
		labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return labels, nil
}
