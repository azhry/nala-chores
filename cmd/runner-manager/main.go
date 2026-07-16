package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/azhry/nala-chores/internal/api"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := api.Config{
		Namespace:   env("RUNNER_NAMESPACE", "agent-runner"),
		Image:       env("RUNNER_BACKEND_IMAGE", "opencode-runner-backend:local"),
		SecretName:  env("RUNNER_SECRET_NAME", "runner-secrets"),
		KubectlPath: env("KUBECTL", "kubectl"),
		ApplyJobs:   envBool("RUNNER_APPLY_JOBS", true),
		StatePath:   env("RUNNER_STATE_PATH", "/tmp/nala-chores-state.json"),
	}

	server := &http.Server{
		Addr:              env("RUNNER_MANAGER_ADDR", ":8080"),
		Handler:           api.NewServer(cfg, log).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		log.Info("runner-manager listening", "addr", server.Addr, "apply_jobs", cfg.ApplyJobs)
		errs <- server.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server stopped", "error", err)
			os.Exit(1)
		}
	case <-sig:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
