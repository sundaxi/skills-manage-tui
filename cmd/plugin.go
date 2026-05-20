package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/plugin"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage agent plugins",
	Long:  `Install, list, and remove agent plugins from GitHub repositories.`,
}

var pluginAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a marketplace from a GitHub repository",
	Long:  `Clone a GitHub repository as a plugin marketplace. Example: skill-tui plugin add affaan-m/ECC`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".skill-tui")
		store := plugin.NewStore(configDir, cfg.PluginsPath)

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		mp, err := store.AddByRepo(ctx, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Marketplace %q added successfully\n", mp.Name)
		fmt.Printf("  Description: %s\n", mp.Description)
		fmt.Printf("  Plugins:     %d\n", len(mp.Plugins))
		fmt.Printf("  Path:        %s\n", store.PluginDir(mp.Name))
		return nil
	},
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugin marketplaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".skill-tui")
		cfg, _ := config.Load()
		pluginsPath := ""
		if cfg != nil {
			pluginsPath = cfg.PluginsPath
		}
		store := plugin.NewStore(configDir, pluginsPath)

		marketplaces, err := store.ScanMarketplaces()
		if err != nil {
			return err
		}

		if len(marketplaces) == 0 {
			fmt.Println("No marketplaces installed. Use 'skill-tui plugin add <owner/repo>' to add one.")
			return nil
		}

		for _, mp := range marketplaces {
			pluginCount := len(mp.Plugins)
			if pluginCount == 0 {
				pluginCount = 1
			}
			fmt.Printf("  %-30s %d plugins  %s\n", mp.Name, pluginCount, mp.Description)
		}
		return nil
	},
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed plugin marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".skill-tui")
		cfg, _ := config.Load()
		pluginsPath := ""
		if cfg != nil {
			pluginsPath = cfg.PluginsPath
		}
		store := plugin.NewStore(configDir, pluginsPath)

		if err := store.RemoveMarketplace(args[0]); err != nil {
			return err
		}
		fmt.Printf("Marketplace %q removed\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginAddCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
}
