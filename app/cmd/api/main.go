// Command api starts the cards REST API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/secrets"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/config"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/logging"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/ports"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/service"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	awsSecrets, err := secrets.NewAWSClient(ctx, os.Getenv("AWS_REGION"))
	if err != nil {
		return err
	}
	cfg, err := config.Load(ctx, os.Getenv, awsSecrets)
	if err != nil {
		return err
	}

	instance := cfg.PodName
	if instance == "" {
		if instance, err = os.Hostname(); err != nil {
			return err
		}
	}

	logger := logging.New(os.Stdout, logging.Options{
		Level:    cfg.LogLevel,
		Format:   cfg.LogFormat,
		Env:      cfg.Env,
		Instance: instance,
	})
	slog.SetDefault(logger)

	db, err := repository.OpenDB(cfg, logger)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	application, err := service.NewApplication(cfg, db)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.Port),
		Handler:           ports.NewHandler(application, instance, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	logger.Info("api listening", "addr", server.Addr)

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
