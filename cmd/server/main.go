package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haowen-xu/agent-coder/internal/app"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	ctx := context.Background()
	application, err := app.New(ctx, *configPath)
	if err != nil {
		slog.Error("start app failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			application.Logger.Error("close app failed", slog.Any("error", closeErr))
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- application.Server.Run()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			application.Logger.Error("server stopped unexpectedly", slog.Any("error", err))
			os.Exit(1)
		}
	case sig := <-sigCh:
		application.Logger.Info("received shutdown signal", slog.String("signal", sig.String()))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), application.Config.Server.ShutdownTimeoutDuration())
		defer cancel()
		if err := application.Server.Shutdown(shutdownCtx); err != nil {
			application.Logger.Error("graceful shutdown failed", slog.Any("error", err))
			os.Exit(1)
		}
	}

	// Give logger/IO a short flush window on shutdown.
	time.Sleep(50 * time.Millisecond)
}
