package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func resetViper() {
	viper.Reset()
	viper.SetEnvPrefix("MONOCRON")
	viper.AutomaticEnv()
}

func execute(args []string) (*cobra.Command, string, error) {
	root := NewRootCommand()
	root.SetArgs(args)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	err := root.Execute()
	return root, buf.String(), err
}

func TestCommandTree(t *testing.T) {
	resetViper()
	root := NewRootCommand()
	expected := map[string]bool{
		"login":     true,
		"logout":    true,
		"config":    true,
		"schedule":  true,
		"runner":    true,
		"execution": true,
		"audit":     true,
		"update":    true,
		"init":      true,
	}
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}
	for name := range expected {
		if !got[name] {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestConfigPathRespectsFlag(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.json")

	_, out, err := execute([]string{"--config", path, "config", "path"})
	if err != nil {
		t.Fatalf("execute: %v\noutput:\n%s", err, out)
	}
	if strings.TrimSpace(out) != path {
		t.Fatalf("expected %q, got %q", path, strings.TrimSpace(out))
	}
}

func TestConfigShowCreatesDefault(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	_, out, err := execute([]string{"--config", path, "config", "show"})
	if err != nil {
		t.Fatalf("execute: %v\noutput:\n%s", err, out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if !strings.Contains(out, DefaultControllerURL) {
		t.Fatalf("output missing default URL: %s", out)
	}
	if !strings.Contains(out, "***") {
		t.Fatalf("output should mask api_key: %s", out)
	}
}

func TestHelpDoesNotCreateConfig(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	_, _, err := execute([]string{"--config", path, "help"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("help should not create config file")
	}
}

func TestCompletionDoesNotCreateConfig(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	_, _, err := execute([]string{"--config", path, "completion", "bash"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("completion should not create config file")
	}
}

func TestLoginWritesConfig(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	_, _, err := execute([]string{"--config", path, "login", "http://example", "my-key"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.ControllerURL != "http://example" {
		t.Fatalf("expected controller_url http://example, got %q", cfg.ControllerURL)
	}
	if cfg.APIKey != "my-key" {
		t.Fatalf("expected api_key my-key, got %q", cfg.APIKey)
	}
}

func TestLogoutClearsAPIKey(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := WriteConfig(path, Config{ControllerURL: "http://example", APIKey: "my-key"}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, _, err := execute([]string{"--config", path, "logout"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.ControllerURL != "http://example" {
		t.Fatalf("controller_url changed unexpectedly")
	}
	if cfg.APIKey != "" {
		t.Fatalf("expected api_key cleared, got %q", cfg.APIKey)
	}
}

func TestMissingControllerURL(t *testing.T) {
	resetViper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	_, _, err := execute([]string{"--config", path, "--controller-url", "", "schedule", "list"})
	if err == nil {
		t.Fatal("expected error for missing controller URL")
	}
	if !strings.Contains(err.Error(), "controller URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
