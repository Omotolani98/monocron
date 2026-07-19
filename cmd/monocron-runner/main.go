package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Omotolani98/monocron/internal/platform/logging"
	"github.com/Omotolani98/monocron/internal/platform/version"
	runnerconfig "github.com/Omotolani98/monocron/internal/runner/config"
	"github.com/Omotolani98/monocron/internal/runner/controllerclient"
	"github.com/Omotolani98/monocron/internal/runner/daemonclient"
	"github.com/Omotolani98/monocron/internal/runner/reconcile"
)

func main() {
	logging.SetDefault(os.Getenv("MONOCRON_LOG_LEVEL"))
	log := slog.Default()
	log.Info("starting monocron-runner", "version", version.Version)

	cfgPath := os.Getenv("MONOCRON_RUNNER_CONFIG")
	if cfgPath == "" {
		cfgPath = runnerconfig.DefaultConfigPath()
	}
	cfg, err := runnerconfig.Load(cfgPath)
	if err != nil {
		log.Error("configuration", "error", err)
		os.Exit(1)
	}

	store := controllerclient.NewTokenStore(cfg.StatePath)
	state, err := store.Load()
	if err != nil {
		log.Error("load runner state", "error", err)
		log.Info("hint: run monocronctl runner join <token> on this machine first")
		os.Exit(1)
	}

	controller := controllerclient.New(cfg.ControllerURL, state.AccessToken)
	daemon := daemonclient.New(cfg.DaemonSocket)

	runner := reconcile.NewRunner(controller, daemon, log, reconcile.RunnerConfig{
		RunnerID:          state.RunnerID,
		Version:           version.Version,
		DaemonVersion:     version.Version,
		Labels:            cfg.Labels,
		PollInterval:      cfg.PollInterval,
		HeartbeatInterval: cfg.HeartbeatInterval,
	})

	runner.Start(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("stopping monocron-runner")
	runner.Stop()
	time.Sleep(100 * time.Millisecond)
	log.Info("monocron-runner stopped")
}
