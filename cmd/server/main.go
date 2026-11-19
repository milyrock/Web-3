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

	"developer.dev/web-3/internal/config"
	"developer.dev/web-3/internal/handlers"
	"developer.dev/web-3/internal/repository"
	"developer.dev/web-3/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	if _, err := os.Stat(cfg.WebDir + "/index.html"); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := repository.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer repo.Close()
	service := usecase.New(repo)
	if cfg.AdminPassword != "" {
		if err := service.EnsureAdmin(startup, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			return err
		}
	}
	server := &http.Server{
		Addr: cfg.Addr, Handler: handlers.New(service, cfg),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	logger.Info("developer.dev started", "address", cfg.Addr)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			server.Close()
			return err
		}
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
