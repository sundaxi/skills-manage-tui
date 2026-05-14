package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
)

var (
	version   = "0.1.0"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "skill-tui [command]",
	Short: "Manage AI coding agent skills across multiple platforms",
	Long: `skill-tui - A CLI tool for managing multi-agent skills.

Manages skills in ~/.agents/skills/ (central registry) and installs
them to 28+ AI coding platforms via symlinks.

Run without arguments to enter interactive TUI mode.`,
	Version: fmt.Sprintf("%s (built %s)", version, buildDate),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		return runTUI(cfg)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(`skill-tui {{.Version}}
`)
}
