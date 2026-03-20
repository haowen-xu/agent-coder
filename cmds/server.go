package cmds

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/haowen-xu/agent-coder/internal/app"
)

// serverCmd 执行相关逻辑。
func serverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()
			application, err := app.New(ctx, configPath)
			if err != nil {
				return err
			}
			defer func() { _ = application.Close() }()

			errCh := make(chan error, 1)
			go func() {
				errCh <- application.Server.Run()
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			select {
			case err := <-errCh:
				return err
			case <-sigCh:
				shutdownCtx, cancel := context.WithTimeout(context.Background(), application.Config.Server.ShutdownTimeoutDuration())
				defer cancel()
				return application.Server.Shutdown(shutdownCtx)
			}
		},
	}
}
