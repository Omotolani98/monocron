package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/Omotolani98/monocron/internal/contracts"
)

// BoundedWriter captures output up to a byte limit, keeping the most recent bytes.
type BoundedWriter struct {
	limit int
	buf   []byte
	full  bool
}

func NewBoundedWriter(limit int) *BoundedWriter {
	return &BoundedWriter{limit: limit}
}

func (b *BoundedWriter) Write(p []byte) (int, error) {
	if len(p) >= b.limit {
		b.buf = append([]byte{}, p[len(p)-b.limit:]...)
		b.full = true
		return len(p), nil
	}
	if len(b.buf)+len(p) <= b.limit {
		b.buf = append(b.buf, p...)
	} else {
		over := len(b.buf) + len(p) - b.limit
		b.buf = append(b.buf[over:], p...)
		b.full = true
	}
	return len(p), nil
}

func (b *BoundedWriter) Bytes() []byte {
	if b.full {
		out := make([]byte, 0, len(b.buf)+len(truncatedMarker))
		out = append(out, truncatedMarker...)
		out = append(out, b.buf...)
		return out
	}
	return b.buf
}

var truncatedMarker = []byte("[...truncated...]\n")

// Result describes the outcome of running a command.
type Result struct {
	State        contracts.ExecutionState
	ExitCode     *int
	ErrorMessage string
	Stdout       []byte
	Stderr       []byte
}

// Run executes argv with the provided context, capturing bounded output.
func Run(ctx context.Context, argv []string, logLimit int) Result {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return Result{
			State:        contracts.ExecutionStateFailed,
			ErrorMessage: "empty command",
		}
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	stdout := NewBoundedWriter(logLimit)
	stderr := NewBoundedWriter(logLimit)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		return Result{
			State:        contracts.ExecutionStateFailed,
			ErrorMessage: fmt.Sprintf("start %q: %v", argv[0], err),
			Stdout:       stdout.Bytes(),
			Stderr:       stderr.Bytes(),
		}
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		killProcessTree(cmd)
		<-done
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = tail(stdout.Bytes(), 4096)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Result{
				State:        contracts.ExecutionStateTimedOut,
				ErrorMessage: msg,
				Stdout:       stdout.Bytes(),
				Stderr:       stderr.Bytes(),
			}
		}
		return Result{
			State:        contracts.ExecutionStateCanceled,
			ErrorMessage: msg,
			Stdout:       stdout.Bytes(),
			Stderr:       stderr.Bytes(),
		}

	case err := <-done:
		if err == nil {
			return Result{
				State:  contracts.ExecutionStateSucceeded,
				Stdout: stdout.Bytes(),
				Stderr: stderr.Bytes(),
			}
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code := exit.ExitCode()
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = tail(stdout.Bytes(), 4096)
			}
			return Result{
				State:        contracts.ExecutionStateFailed,
				ExitCode:     &code,
				ErrorMessage: msg,
				Stdout:       stdout.Bytes(),
				Stderr:       stderr.Bytes(),
			}
		}
		return Result{
			State:        contracts.ExecutionStateFailed,
			ErrorMessage: fmt.Sprintf("wait failed: %v", err),
			Stdout:       stdout.Bytes(),
			Stderr:       stderr.Bytes(),
		}
	}
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		_ = cmd.Process.Kill()
	}
}

func tail(b []byte, n int) string {
	if len(b) <= n {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(string(b[len(b)-n:]))
}

func (b *BoundedWriter) String() string {
	return string(b.Bytes())
}

// CopyFrom copies from r into the bounded writer up to the configured limit.
func (b *BoundedWriter) CopyFrom(r io.Reader) (int64, error) {
	return io.Copy(b, r)
}
