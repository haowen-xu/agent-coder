package cmds

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "agent-coder",
		Short: "Agent Coder CLI",
	}
)

// Execute 执行相关逻辑。
func Execute() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to config file")
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(workerCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(syncIssuesCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "command failed: %v\n", err)
		os.Exit(1)
	}
}
