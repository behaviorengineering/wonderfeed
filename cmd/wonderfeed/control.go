package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/behaviorengineering/wonderfeed/internal/controlplane/store"
	"github.com/behaviorengineering/wonderfeed/internal/httpapi"
	"github.com/behaviorengineering/wonderfeed/internal/provider/ytzero"
)

func runControl(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printControlUsage(stderr)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		printControlUsage(stdout)
		return 0
	case "serve":
		return runControlServe(args[1:], stdout, stderr)
	case "migrate":
		return runControlMigrate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown control command: %s\n\n", args[0])
		printControlUsage(stderr)
		return 2
	}
}

func printControlUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: wonderfeed control <command>

Commands:
  serve        Start the parent control-plane HTTP API
  migrate up   Apply host control-plane database migrations

Environment:
  WONDERFEED_CONTROL_BIND     Listen address (default 127.0.0.1:8080)
  DATABASE_URL                PostgreSQL DSN (required)
  YTZERO_BASE_URL             Provider base URL (default http://127.0.0.1:3001)
  YTZERO_SESSION_COOKIE       Optional Cookie header for provider API auth
  WONDERFEED_PARENT_AUTH_KEY  Required for non-loopback binds

`)
}

func runControlMigrate(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] != "up" {
		fmt.Fprintln(stderr, "usage: wonderfeed control migrate up")
		return 2
	}
	cfg := loadControlConfig()
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		fmt.Fprintln(stderr, "control migrate: DATABASE_URL is required")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "control migrate: %v\n", err)
		return 1
	}
	defer pool.Close()
	if err := store.MigrateUp(ctx, pool); err != nil {
		fmt.Fprintf(stderr, "control migrate: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Control-plane migrations applied.")
	return 0
}

func runControlServe(args []string, stdout, stderr io.Writer) int {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			fmt.Fprint(stdout, `Usage: wonderfeed control serve

Starts the parent control-plane HTTP API. Binds to WONDERFEED_CONTROL_BIND
(default 127.0.0.1:8080). Non-loopback binds require WONDERFEED_PARENT_AUTH_KEY.

`)
			return 0
		}
	}

	cfg := loadControlConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "control serve: %v\n", err)
		return 2
	}

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer bootCancel()

	pool, err := store.OpenPool(bootCtx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "control serve: %v\n", err)
		return 1
	}
	defer pool.Close()

	if err := store.MigrateUp(bootCtx, pool); err != nil {
		fmt.Fprintf(stderr, "control serve: migrate: %v\n", err)
		return 1
	}

	logger := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	st := store.NewPostgresStore(pool, time.Now)
	adapter := ytzero.NewAdapter(ytzero.AdapterConfig{
		BaseURL:       cfg.ProviderBaseURL,
		SessionCookie: cfg.ProviderSessionCookie,
		HTTP:          http.DefaultClient,
	})
	svc := controlplane.NewService(controlplane.ServiceConfig{
		Store:    st,
		Provider: adapter,
		Clock:    time.Now,
		Logger:   logger,
	})
	handler := httpapi.NewHandler(httpapi.HandlerConfig{
		Service:       svc,
		ParentAuthKey: cfg.ParentAuthKey,
	})
	mux := http.NewServeMux()
	handler.Mount(mux)

	server := &http.Server{
		Addr:              cfg.Bind,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("control plane listening", "bind", cfg.Bind)
		fmt.Fprintf(stdout, "Control plane listening on http://%s\n", cfg.Bind)
		errCh <- server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(stderr, "control serve: %v\n", err)
			return 1
		}
		return 0
	case sig := <-sigCh:
		logger.Info("control plane shutting down", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(stderr, "control serve: shutdown: %v\n", err)
			return 1
		}
		return 0
	}
}

func loadControlConfig() controlplane.Config {
	cfg := controlplane.LoadConfigFromEnv()
	if cfg.DatabaseURL == "" {
		if pw := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD")); pw != "" {
			user := envOrDefault("POSTGRES_USER", "ytzero")
			db := envOrDefault("POSTGRES_DB", "ytzero")
			cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@127.0.0.1:5432/%s?sslmode=disable", user, pw, db)
		}
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
