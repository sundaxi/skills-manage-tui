package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
	Long:  `View current configuration, set values, and list detected platforms.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			fmt.Printf("skills_path: %s\n", cfg.SkillsPath)
			fmt.Printf("theme: %s\n", cfg.Theme)
			fmt.Printf("language: %s\n", cfg.Language)
			fmt.Printf("github_token: %s\n", maskToken(cfg.GitHubToken))
			fmt.Printf("ai_provider: %s\n", cfg.AIProvider)
			fmt.Printf("ai_key: %s\n", maskToken(cfg.AIKey))
			fmt.Printf("config_file: %s\n", config.ConfigPath())
			return nil
		}

		key := args[0]
		switch key {
		case "skills_path":
			fmt.Println(cfg.SkillsPath)
		case "theme":
			fmt.Println(cfg.Theme)
		case "language":
			fmt.Println(cfg.Language)
		case "github_token":
			fmt.Println(maskToken(cfg.GitHubToken))
		case "ai_provider":
			fmt.Println(cfg.AIProvider)
		case "ai_key":
			fmt.Println(maskToken(cfg.AIKey))
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		key, value := args[0], args[1]
		switch key {
		case "skills_path":
			cfg.SkillsPath = value
		case "theme":
			cfg.Theme = value
		case "language":
			cfg.Language = value
		case "github_token":
			cfg.GitHubToken = value
		case "ai_provider":
			cfg.AIProvider = value
		case "ai_key":
			cfg.AIKey = value
		case "ai_endpoint":
			cfg.AIEndpoint = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("Set %s = %s\n", key, maskIfSecret(key, value))
		return nil
	},
}

var configPlatformsCmd = &cobra.Command{
	Use:   "platforms",
	Short: "List all known platforms and their status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Printf("%-20s %-10s %-10s %s\n", "NAME", "CATEGORY", "INSTALLED", "SKILLS DIR")
		for _, p := range cfg.Platforms {
			installed := "no"
			if dirExists(p.SkillsDir) {
				installed = "yes"
			}
			fmt.Printf("%-20s %-10s %-10s %s\n", p.Name, p.Category, installed, p.SkillsDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPlatformsCmd)
}

func maskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func maskIfSecret(key, value string) string {
	switch key {
	case "github_token", "ai_key":
		return maskToken(value)
	default:
		return value
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
