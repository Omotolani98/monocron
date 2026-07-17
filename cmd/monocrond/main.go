package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Omotolani98/monocron/internal/daemon/api"
	"github.com/Omotolani98/monocron/internal/daemon/scheduler"
	"github.com/Omotolani98/monocron/internal/daemon/store"
	"github.com/Omotolani98/monocron/internal/platform/config"
	"github.com/Omotolani98/monocron/internal/platform/logging"
	"github.com/Omotolani98/monocron/internal/platform/version"
)

func main() {
	logging.SetDefault(os.Getenv("MONOCRON_LOG_LEVEL"))
	log := slog.Default()
	log.Info("starting monocrond", "version", version.Version)

	cfg, err := config.DaemonConfigFromEnv()
	if err != nil {
		log.Error("configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.StatePath)
	if err != nil {
		log.Error("open store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	sched := scheduler.New(log, st, cfg.MaxLogBytes)
	if cfg.DefaultTimeout > 0 {
		// Default timeout is applied inside the scheduler when an assignment has none.
	}
	sched.Start()

	// Reload persisted assignments so schedules survive restarts.
	assignments, err := st.LoadAssignments(ctx)
	if err != nil {
		log.Error("load assignments", "error", err)
		os.Exit(1)
	}
	for _, a := range assignments {
		if err := sched.Add(ctx, a); err != nil {
			log.Error("restore assignment", "assignment_id", a.ID, "error", err)
		}
	}
	log.Info("restored assignments", "count", len(assignments))

	srv := api.NewServer(cfg.SocketPath, sched, st, log)
	if err := srv.Listen(); err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case sig := <-sigCh:
		log.Info("received signal", "signal", sig)
	case err := <-errCh:
		log.Error("server error", "error", err)
	}

	stopCtx := sched.Stop()
	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Close(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
	}
	log.Info("monocrond stopped")
}
