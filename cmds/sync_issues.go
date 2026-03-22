package main

import (
	"context"

	"github.com/spf13/cobra"
)

// syncIssuesCmd 执行相关逻辑。
func syncIssuesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync-issues",
		Short: "Run one-shot issue sync + scheduling tick",
		RunE: func(_ *cobra.Command, _ []string) error {
			application, err := newApplication(context.Background(), configPath)
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
			return runWorkerOnce(wk, context.Background())
		},
	}
}
