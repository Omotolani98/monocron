package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %q", path)
	}
	if filepath.Base(path) != "config.json" {
		t.Fatalf("expected config.json, got %q", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != "monocronctl" {
		t.Fatalf("expected monocronctl directory, got %q", filepath.Base(filepath.Dir(path)))
	}
}

func TestCreateDefaultIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := CreateDefaultIfMissing(path); err != nil {
		t.Fatalf("create default config: %v", err)
	}

	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.ControllerURL != DefaultControllerURL {
		t.Fatalf("expected controller_url %q, got %q", DefaultControllerURL, cfg.ControllerURL)
	}
	if cfg.APIKey != "" {
		t.Fatalf("expected empty api_key, got %q", cfg.APIKey)
	}

	// Second call must be idempotent.
	if err := CreateDefaultIfMissing(path); err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	cfg2, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("read config after second create: %v", err)
	}
	if cfg2.ControllerURL != cfg.ControllerURL {
		t.Fatalf("config changed after idempotent create")
	}
}

func TestWriteConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "monocronctl", "config.json")

	if err := WriteConfig(path, Config{ControllerURL: "http://example", APIKey: "secret"}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config file is too permissive: %o", info.Mode().Perm())
	}

	cfgDir := filepath.Dir(path)
	dirInfo, err := os.Stat(cfgDir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if dirInfo.Mode().Perm()&0o077 != 0 {
		t.Fatalf("config dir is too permissive: %o", dirInfo.Mode().Perm())
	}
}

func TestReadConfigReturnsParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if _, err := ReadConfig(path); err == nil {
		t.Fatal("expected parse error")
	}
}
