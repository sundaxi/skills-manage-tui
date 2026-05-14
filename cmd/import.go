package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/github"
)

var importCmd = &cobra.Command{
	Use:   "import <github-url>",
	Short: "Import skills from a GitHub repository",
	Long: `Import skills from a GitHub repository URL.

Scans the repository for SKILL.md files and imports them into the central registry.
Optionally specify a subdirectory with --path.

Set SKILL_CLI_GITHUB_TOKEN or configure via 'skill-tui config set github_token <token>'
for higher rate limits.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		repoURL := args[0]
		subPath, _ := cmd.Flags().GetString("path")

		client := github.NewClient(cfg.GitHubToken)
		importer := github.NewImporter(client, cfg.SkillsPath)

		fmt.Printf("Importing from %s", repoURL)
		if subPath != "" {
			fmt.Printf(" (path: %s)", subPath)
		}
		fmt.Println("...")

		results, err := importer.ImportFromURL(context.Background(), repoURL, subPath)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		imported := 0
		for _, r := range results {
			if r.Skipped {
				fmt.Printf("  Skipped %s (error reading content)\n", r.SkillName)
				continue
			}
			fmt.Printf("  Imported %s → %s\n", r.SkillName, r.Path)
			imported++
		}

		fmt.Printf("\nDone: %d skill(s) imported.\n", imported)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("path", "p", "", "Subdirectory path within the repository")
}
