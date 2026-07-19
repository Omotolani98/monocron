package cli

import (
	"testing"

	"github.com/Omotolani98/monocron/internal/runner/config"
	"github.com/spf13/cobra"
)

func TestResolveStatePathFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "runner"}
	cmd.PersistentFlags().String("state", "", "")
	if err := cmd.PersistentFlags().Set("state", "/flag.state"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	got, err := resolveStatePath(cmd)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "/flag.state" {
		t.Fatalf("expected flag value, got %q", got)
	}
}

func TestResolveStatePathEnv(t *testing.T) {
	t.Setenv("MONOCRON_RUNNER_STATE_PATH", "/env.state")
	cmd := &cobra.Command{Use: "runner"}
	cmd.PersistentFlags().String("state", "", "")
	got, err := resolveStatePath(cmd)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "/env.state" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestResolveStatePathDefault(t *testing.T) {
	t.Setenv("MONOCRON_RUNNER_STATE_PATH", "")
	cmd := &cobra.Command{Use: "runner"}
	cmd.PersistentFlags().String("state", "", "")
	got, err := resolveStatePath(cmd)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != config.DefaultStatePath() {
		t.Fatalf("expected default state path, got %q", got)
	}
}
