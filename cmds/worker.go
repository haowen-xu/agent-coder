package main

import (
	"context"
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

// workerCmd 执行相关逻辑。
func workerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Start scheduler + run executor worker",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			application, err := newApplication(ctx, configPath)
			if err != nil {
				return err
			}
			defer func() { _ = application.Close() }()

			wk := newWorkerSvc(
				application.Config,
				application.Logger,
				application.DB,
				application.PromptStore,
				application.Secret,
			)

			errCh := make(chan error, 1)
			runWorkerLoopFn := runWorkerLoop
			go func() {
				errCh <- runWorkerLoopFn(wk, ctx)
			}()

			sigCh := make(chan os.Signal, 1)
			notifySignals(sigCh, syscall.SIGINT, syscall.SIGTERM)
			defer stopSignals(sigCh)

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
