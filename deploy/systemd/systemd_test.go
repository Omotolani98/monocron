package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readService(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func TestControllerService(t *testing.T) {
	content := readService(t, "monocron-controller.service")
	if !strings.Contains(content, "EnvironmentFile=-/etc/monocron/controller.env") {
		t.Error("missing controller.env EnvironmentFile")
	}
	if !strings.Contains(content, "ExecStart=/usr/local/bin/monocron-controller") {
		t.Error("missing ExecStart")
	}
	if strings.Contains(content, "ctrl.example.com") {
		t.Error("service contains placeholder URL")
	}
}

func TestDaemonService(t *testing.T) {
	content := readService(t, "monocrond.service")
	if !strings.Contains(content, "ExecStart=/usr/local/bin/monocrond") {
		t.Error("missing ExecStart")
	}
	if !strings.Contains(content, "RuntimeDirectory=monocron") {
		t.Error("missing RuntimeDirectory")
	}
	if !strings.Contains(content, "StateDirectory=monocron") {
		t.Error("missing StateDirectory")
	}
}

func TestRunnerService(t *testing.T) {
	content := readService(t, "monocron-runner.service")
	if !strings.Contains(content, "EnvironmentFile=-/etc/monocron/runner.env") {
		t.Error("missing runner.env EnvironmentFile")
	}
	if !strings.Contains(content, "Requires=monocrond.service") {
		t.Error("missing dependency on monocrond")
	}
	if !strings.Contains(content, "ExecStart=/usr/local/bin/monocron-runner") {
		t.Error("missing ExecStart")
	}
	if strings.Contains(content, "ctrl.example.com") {
		t.Error("service contains placeholder URL")
	}
}

func TestControllerEnvExample(t *testing.T) {
	content := readService(t, "controller.env")
	if !strings.Contains(content, "MONOCRON_DATABASE_URL=") {
		t.Error("env example missing MONOCRON_DATABASE_URL")
	}
}
