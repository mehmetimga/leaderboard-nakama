package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-campions/leaderboard-nakama/internal/config"
	httpPkg "github.com/ai-campions/leaderboard-nakama/internal/http"
	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
	"github.com/ai-campions/leaderboard-nakama/internal/nakama"
	"github.com/ai-campions/leaderboard-nakama/internal/postgres"
	"github.com/ai-campions/leaderboard-nakama/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Nakama client
	nakamaClient := nakama.NewClient(cfg.Nakama)

	// Initialize PostgreSQL repository
	repo, err := postgres.NewRepository(ctx, cfg.Postgres)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer repo.Close()

	// Run database migrations
	if err := repo.RunMigrations(ctx); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("database migrations completed")

	// Initialize leaderboard service
	service := leaderboard.NewService(nakamaClient, repo)

	// Initialize HTTP handler and router
	handler := httpPkg.NewHandler(service)
	router := httpPkg.NewRouter(handler)

	// Initialize and start snapshot worker
	var snapshotWorker *worker.SnapshotWorker
	if cfg.Worker.Enabled {
		snapshotWorker = worker.NewSnapshotWorker(service, repo, cfg.Worker)

		// Register default leaderboards to snapshot
		// In production, this could be configured via API or config file
		defaultLeaderboards := os.Getenv("SNAPSHOT_LEADERBOARDS")
		if defaultLeaderboards != "" {
			// Parse comma-separated list
			for _, lb := range splitAndTrim(defaultLeaderboards) {
				snapshotWorker.RegisterLeaderboard(lb)
			}
		}

		snapshotWorker.Start(ctx)
		defer snapshotWorker.Stop()
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("starting server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range split(s, ',') {
		trimmed := trim(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func split(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

