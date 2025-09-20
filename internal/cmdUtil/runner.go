package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	maxLogBytes = 256 * 1024
)

type limitedBuffer struct {
	buf  []byte
	cap  int
	full bool
}

func newLimitedBuffer(n int) *limitedBuffer { return &limitedBuffer{cap: n} }

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) >= l.cap {
		l.buf = append([]byte{}, p[len(p)-l.cap:]...)
		l.full = true
		return len(p), nil
	}
	if len(l.buf)+len(p) <= l.cap {
		l.buf = append(l.buf, p...)
	} else {
		over := len(l.buf) + len(p) - l.cap
		l.buf = append(l.buf[over:], p...)
		l.full = true
	}
	return len(p), nil
}

func (l *limitedBuffer) Bytes() []byte { return l.buf }
func (l *limitedBuffer) String() string {
	s := string(l.buf)
	if l.full {
		return "[…truncated…]\n" + s
	}
	return s
}

func RunCommand(ctx context.Context, argv []string) error {
	if len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
		return errors.New("runCommand: empty argv")
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	stdout := newLimitedBuffer(maxLogBytes)
	stderr := newLimitedBuffer(maxLogBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %q: %w", argv[0], err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		killProcessTree(cmd)
		<-done
		if dl, ok := ctx.Deadline(); ok && time.Now().After(dl) {
			return fmt.Errorf("timeout: %s", strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("canceled: %s", strings.TrimSpace(stderr.String()))

	case err := <-done:
		if err == nil {
			return nil
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code := exit.ExitCode()
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = tail(stdout.Bytes(), 4<<10) // 4 KiB tail
			}
			return fmt.Errorf("exit %d: %s", code, msg)
		}
		return fmt.Errorf("wait failed: %w", err)
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
