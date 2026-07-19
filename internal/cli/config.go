package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultControllerURL is the default controller endpoint.
const DefaultControllerURL = "http://127.0.0.1:8080"

// Config holds persisted monocronctl configuration.
type Config struct {
	ControllerURL string `json:"controller_url" mapstructure:"controller_url"`
	APIKey        string `json:"api_key" mapstructure:"api_key"`
}

// DefaultConfigPath returns the platform-specific user config directory path.
// It falls back to ~/.config when the OS cannot provide a config directory.
func DefaultConfigPath() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "monocronctl", "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "monocronctl", "config.json")
}

// EnsureConfigDir creates the parent directory for path with restricted permissions.
func EnsureConfigDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

// ReadConfig reads a config file from disk.
func ReadConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// WriteConfig writes cfg to path atomically with 0600 permissions.
func WriteConfig(path string, cfg Config) error {
	if err := EnsureConfigDir(path); err != nil {
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

// CreateDefaultIfMissing creates a default config file when none exists.
// It is idempotent and never overwrites an existing file.
func CreateDefaultIfMissing(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return WriteConfig(path, Config{ControllerURL: DefaultControllerURL})
}
