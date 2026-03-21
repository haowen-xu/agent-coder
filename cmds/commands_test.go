package main

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandBuilders 用于单元测试。
func TestCommandBuilders(t *testing.T) {
	t.Parallel()

	if cmd := serverCmd(); cmd == nil || cmd.Use != "server" {
		t.Fatalf("serverCmd mismatch: %#v", cmd)
	}
	if cmd := workerCmd(); cmd == nil || cmd.Use != "worker" {
		t.Fatalf("workerCmd mismatch: %#v", cmd)
	}
	if cmd := migrateCmd(); cmd == nil || cmd.Use != "migrate" {
		t.Fatalf("migrateCmd mismatch: %#v", cmd)
	}
	if cmd := syncIssuesCmd(); cmd == nil || cmd.Use != "sync-issues" {
		t.Fatalf("syncIssuesCmd mismatch: %#v", cmd)
	}
}

// TestCommandRunEConfigError 用于单元测试。
func TestCommandRunEConfigError(t *testing.T) {
	oldPath := configPath
	defer func() { configPath = oldPath }()
	configPath = "/tmp/definitely-not-exists/config.yaml"

	commands := []string{"server", "worker", "migrate", "sync-issues"}
	for _, name := range commands {
		name := name
		t.Run(name, func(t *testing.T) {
			var err error
			switch name {
			case "server":
				cmd := serverCmd()
				err = cmd.RunE(cmd, nil)
			case "worker":
				cmd := workerCmd()
				err = cmd.RunE(cmd, nil)
			case "migrate":
				cmd := migrateCmd()
				err = cmd.RunE(cmd, nil)
			case "sync-issues":
				cmd := syncIssuesCmd()
				err = cmd.RunE(cmd, nil)
			}
			if err == nil {
				t.Fatalf("%s should fail on missing config", name)
			}
		})
	}
}

// TestExecuteAndMainHelp 用于单元测试。
func TestExecuteAndMainHelp(t *testing.T) {
	oldRoot := rootCmd
	oldPath := configPath
	defer func() {
		rootCmd = oldRoot
		configPath = oldPath
	}()

	rootCmd = &cobra.Command{
		Use:   "agent-coder",
		Short: "Agent Coder CLI",
	}
	rootCmd.SetArgs([]string{"--help"})
	Execute()
	if len(rootCmd.Commands()) < 4 {
		t.Fatalf("Execute should register at least 4 subcommands, got: %d", len(rootCmd.Commands()))
	}

	rootCmd = &cobra.Command{
		Use:   "agent-coder",
		Short: "Agent Coder CLI",
	}
	rootCmd.SetArgs([]string{"--help"})
	main()
}
