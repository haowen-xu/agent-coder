package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/haowen-xu/agent-coder/internal/app"
)

// migrateCmd 执行相关逻辑。
func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations (via gorm automigrate)",
		RunE: func(_ *cobra.Command, _ []string) error {
			application, err := app.New(context.Background(), configPath)
			if err != nil {
				return err
			}
			defer func() { _ = application.Close() }()
			return nil
		},
	}
}
