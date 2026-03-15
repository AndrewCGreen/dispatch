package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dispatch-email/dispatch/internal/config"
	"github.com/dispatch-email/dispatch/internal/server"
	"github.com/dispatch-email/dispatch/internal/store"
	"github.com/dispatch-email/dispatch/internal/queue"
	"github.com/dispatch-email/dispatch/internal/backend"
)

func runServe() error {
	cfgPath := "dispatch.yaml"
	if len(os.Args) > 2 {
		cfgPath = os.Args[2]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set up logging
	logLevel := slog.LevelInfo
	if cfg.Logging.Level == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Initialize database
	db, err := store.New(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize backends
	backends, err := backend.InitFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize backends: %w", err)
	}

	// Initialize send queue
	q := queue.New(db, backends, queue.Options{
		Workers:      cfg.Queue.Workers,
		RetryMax:     cfg.Queue.RetryMax,
		RetryBackoff: cfg.Queue.RetryBackoff,
	})

	// Create and start server
	srv := server.New(cfg, db, backends, q, logger)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start queue workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	slog.Info("dispatch starting", "addr", httpServer.Addr, "version", version)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("dispatch stopped")
	return nil
}
