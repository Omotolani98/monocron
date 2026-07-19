package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ControllerConfig holds controller runtime configuration.
type ControllerConfig struct {
	ListenAddr string
	DatabaseURL string
	LogLevel   string
	MetricsAddr string
}

func ControllerConfigFromEnv() (ControllerConfig, error) {
	cfg := ControllerConfig{
		ListenAddr:  getEnv("MONOCRON_CONTROLLER_LISTEN", ":8080"),
		DatabaseURL: os.Getenv("MONOCRON_DATABASE_URL"),
		LogLevel:    getEnv("MONOCRON_LOG_LEVEL", "info"),
		MetricsAddr: getEnv("MONOCRON_METRICS_LISTEN", ":9090"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("MONOCRON_DATABASE_URL is required")
	}
	return cfg, nil
}

// DaemonConfig holds daemon runtime configuration.
type DaemonConfig struct {
	SocketPath    string
	StatePath     string
	LogLevel      string
	MaxLogBytes   int
	DefaultTimeout time.Duration
	MaxConcurrent int
}

func DaemonConfigFromEnv() (DaemonConfig, error) {
	cfg := DaemonConfig{
		SocketPath:     getEnv("MONOCRON_DAEMON_SOCKET", "/run/monocron/monocrond.sock"),
		StatePath:      getEnv("MONOCRON_DAEMON_STATE_PATH", "/var/lib/monocron/monocrond.db"),
		LogLevel:       getEnv("MONOCRON_LOG_LEVEL", "info"),
		MaxLogBytes:    getEnvInt("MONOCRON_MAX_LOG_BYTES", 256*1024),
		DefaultTimeout: parseDuration(getEnv("MONOCRON_DEFAULT_TIMEOUT", "30s")),
		MaxConcurrent:  getEnvInt("MONOCRON_MAX_CONCURRENT", 0),
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func parseDuration(v string) time.Duration {
	if v == "" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

func parseLabels(v string) (map[string]string, error) {
	labels := map[string]string{}
	if v == "" {
		return labels, nil
	}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid label %q, expected key=value", part)
		}
		labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return labels, nil
}
