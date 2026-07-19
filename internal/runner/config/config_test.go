package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultPaths(t *testing.T) {
	if !strings.Contains(DefaultConfigPath(), "monocron/runner.json") {
		t.Fatalf("unexpected default config path: %s", DefaultConfigPath())
	}
	if !strings.Contains(DefaultStatePath(), "monocron/runner.state") {
		t.Fatalf("unexpected default state path: %s", DefaultStatePath())
	}
}

func TestEnsureConfigCreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner.json")

	if err := EnsureConfig(path); err != nil {
		t.Fatalf("ensure config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config not created: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config file too permissive: %o", info.Mode().Perm())
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(b), `"poll_interval": "10s"`) {
		t.Fatalf("expected human-readable duration, got:\n%s", string(b))
	}
}

func TestLoadReturnsDefaultsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner.json")

	t.Setenv("MONOCRON_CONTROLLER_URL", "http://override")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ControllerURL != "http://override" {
		t.Fatalf("expected controller URL override, got %q", cfg.ControllerURL)
	}
	if cfg.PollInterval != 10*time.Second {
		t.Fatalf("expected poll interval 10s, got %v", cfg.PollInterval)
	}
	if cfg.HeartbeatInterval != 10*time.Second {
		t.Fatalf("expected heartbeat interval 10s, got %v", cfg.HeartbeatInterval)
	}
	if cfg.StatePath != DefaultStatePath() {
		t.Fatalf("expected default state path, got %q", cfg.StatePath)
	}
}

func TestLoadUsesRunnerSpecificEnvVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner.json")

	t.Setenv("MONOCRON_RUNNER_STATE_PATH", "/var/lib/monocron/runner.state")
	t.Setenv("MONOCRON_RUNNER_TYPE", "vm")
	t.Setenv("MONOCRON_RUNNER_LABELS", "zone=home,os=linux")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.StatePath != "/var/lib/monocron/runner.state" {
		t.Fatalf("expected state path override, got %q", cfg.StatePath)
	}
	if cfg.Type != "vm" {
		t.Fatalf("expected type override, got %q", cfg.Type)
	}
	if cfg.Labels["zone"] != "home" || cfg.Labels["os"] != "linux" {
		t.Fatalf("expected labels, got %v", cfg.Labels)
	}
}

func TestLoadUsesConfiguredStatePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner.json")
	statePath := filepath.Join(dir, "custom.state")

	cfg := DefaultConfig()
	cfg.StatePath = statePath
	b, _ := cfg.MarshalJSON()
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.StatePath != statePath {
		t.Fatalf("expected state path %q, got %q", statePath, loaded.StatePath)
	}
}

func TestReadConfigParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runner.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadConfig(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseLabels(t *testing.T) {
	got, err := parseLabels("zone=home, os=linux , foo=bar")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]string{"zone": "home", "os": "linux", "foo": "bar"}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("expected %s=%s, got %s", k, v, got[k])
		}
	}
}

func TestParseLabelsInvalid(t *testing.T) {
	if _, err := parseLabels("zone"); err == nil {
		t.Fatal("expected error for invalid label")
	}
}
