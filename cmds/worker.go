package cmds

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/haowen-xu/agent-coder/internal/app"
	"github.com/haowen-xu/agent-coder/internal/service/worker"
)

func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Start scheduler + run executor worker",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			application, err := app.New(ctx, configPath)
			if err != nil {
				return err
			}
			defer func() { _ = application.Close() }()

			wk := worker.New(application.Config, application.Logger, application.DB, application.PromptStore)

			errCh := make(chan error, 1)
			go func() {
				errCh <- wk.RunLoop(ctx)
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			select {
			case err := <-errCh:
				return err
			case <-sigCh:
				cancel()
				return nil
			}
		},
	}
}
