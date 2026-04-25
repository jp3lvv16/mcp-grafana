package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/grafana/mcp-grafana/pkg/server"
)

var (
	// Version is set at build time via ldflags
	Version = "dev"
	// Commit is set at build time via ldflags
	Commit = "unknown"
)

func main() {
	var (
		grafanaURL   string
		grafanaToken string
		transport    string
		addr         string
		logLevel     string
		showVersion  bool
	)

	flag.StringVar(&grafanaURL, "grafana-url", getEnvOrDefault("GRAFANA_URL", "http://localhost:3000"), "Grafana instance URL")
	flag.StringVar(&grafanaToken, "grafana-token", os.Getenv("GRAFANA_API_KEY"), "Grafana API token for authentication")
	flag.StringVar(&transport, "transport", getEnvOrDefault("MCP_TRANSPORT", "stdio"), "Transport type: stdio or sse")
	// Changed default port from 8080 to 8081 to avoid conflicts with other local services on my machine
	flag.StringVar(&addr, "addr", getEnvOrDefault("MCP_ADDR", ":8081"), "Address to listen on (only used with sse transport)")
	// Default to info level to reduce noise in personal usage
	flag.StringVar(&logLevel, "log-level", getEnvOrDefault("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("mcp-grafana version %s (commit: %s)\n", Version, Commit)
		os.Exit(0)
	}

	// Configure structured logging
	logger := setupLogger(logLevel)
	slog.SetDefault(logger)

	slog.Info("starting mcp-grafana",
		"version", Version,
		"commit", Commit,
		"transport", transport,
		"grafana_url", grafanaURL,
	)

	// Validate required configuration
	if grafanaURL == "" {
		slog.Error("grafana-url is required")
		os.Exit(1)
	}

	cfg := &server.Config{
		GrafanaURL:   grafanaURL,
		GrafanaToken: grafanaToken,
		Transport:    transport,
		Addr:         addr,
	}

	// Set up context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(ctx); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}

// setupLogger creates a structured logger based on the specified log level.
func setupLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
}

// getEnvOrDefault returns the value of an environment variable or a default value.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
