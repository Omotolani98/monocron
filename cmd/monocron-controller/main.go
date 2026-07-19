package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Omotolani98/monocron/internal/controller/api"
	"github.com/Omotolani98/monocron/internal/controller/store"
	"github.com/Omotolani98/monocron/internal/platform/config"
	"github.com/Omotolani98/monocron/internal/platform/logging"
	"github.com/Omotolani98/monocron/internal/platform/version"
)

func main() {
	logging.SetDefault(os.Getenv("MONOCRON_LOG_LEVEL"))
	log := slog.Default()
	log.Info("starting monocron-controller", "version", version.Version)

	cfg, err := config.ControllerConfigFromEnv()
	if err != nil {
		log.Error("configuration", "error", err)
		os.Exit(1)
	}

	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	srv := api.NewServer(cfg.ListenAddr, db, log)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Run(); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	exitCode := 0
	select {
	case sig := <-sigCh:
		log.Info("received signal", "signal", sig)
	case err := <-errCh:
		log.Error("server error", "error", err)
		exitCode = 1
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Close(shutdownCtx); err != nil {
		log.Error("shutdown", "error", err)
		exitCode = 1
	}
	log.Info("monocron-controller stopped")
	os.Exit(exitCode)
}
