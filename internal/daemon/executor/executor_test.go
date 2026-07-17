package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Omotolani98/monocron/internal/contracts"
)

func TestRunSuccess(t *testing.T) {
	res := Run(context.Background(), []string{"echo", "hello"}, 1024)
	if res.State != contracts.ExecutionStateSucceeded {
		t.Fatalf("expected succeeded, got %s", res.State)
	}
	if !strings.Contains(string(res.Stdout), "hello") {
		t.Fatalf("expected stdout to contain hello, got %q", string(res.Stdout))
	}
}

func TestRunFailure(t *testing.T) {
	res := Run(context.Background(), []string{"sh", "-c", "exit 7"}, 1024)
	if res.State != contracts.ExecutionStateFailed {
		t.Fatalf("expected failed, got %s", res.State)
	}
	if res.ExitCode == nil || *res.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %v", res.ExitCode)
	}
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	res := Run(ctx, []string{"sleep", "10"}, 1024)
	if res.State != contracts.ExecutionStateTimedOut {
		t.Fatalf("expected timed_out, got %s", res.State)
	}
}

func TestBoundedWriter(t *testing.T) {
	w := NewBoundedWriter(10)
	_, _ = w.Write([]byte("12345"))
	_, _ = w.Write([]byte("67890"))
	_, _ = w.Write([]byte("abc"))
	got := string(w.Bytes())
	if !strings.Contains(got, "[...truncated...]") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
	if !strings.HasSuffix(got, "890abc") {
		t.Fatalf("expected to keep last bytes, got %q", got)
	}
}
