package main

import (
	"context"
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

// serverCmd 执行相关逻辑。
func serverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := ensureWebUIAssets(); err != nil {
				return err
			}
			ctx := context.Background()
			application, err := newApplication(ctx, configPath)
			if err != nil {
				return err
			}
			defer func() { _ = application.Close() }()

			errCh := make(chan error, 1)
			runServerFn := runServer
			go func() {
				errCh <- runServerFn(application)
			}()

			sigCh := make(chan os.Signal, 1)
			notifySignals(sigCh, syscall.SIGINT, syscall.SIGTERM)
			defer stopSignals(sigCh)

			select {
			case err := <-errCh:
				return err
			case <-sigCh:
				shutdownCtx, cancel := context.WithTimeout(context.Background(), application.Config.Server.ShutdownTimeoutDuration())
				defer cancel()
				return shutdownServer(application, shutdownCtx)
			}
		},
	}
}
