package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/github"
	"github.com/ying-sun1/skill-tui/internal/marketplace"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse and install skills from the marketplace",
	Long:  `Browse the skill marketplace, search for skills, and install them directly.`,
}

var marketplaceBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse marketplace publishers and skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := marketplace.NewClient()

		publishers, err := client.ListPublishers(context.Background())
		if err != nil {
			return fmt.Errorf("fetching publishers: %w", err)
		}

		if len(publishers) == 0 {
			fmt.Println("No publishers found in the marketplace.")
			fmt.Println("Check your internet connection or try again later.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PUBLISHER\tSKILLS\tDESCRIPTION")
		for _, p := range publishers {
			fmt.Fprintf(w, "%s\t%d\t%s\n", p.Name, p.SkillCount, p.Description)
		}
		w.Flush()

		return nil
	},
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the marketplace for skills",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := marketplace.NewClient()

		results, err := client.Search(context.Background(), args[0])
		if err != nil {
			return fmt.Errorf("searching marketplace: %w", err)
		}

		if len(results) == 0 {
			fmt.Printf("No skills found matching %q\n", args[0])
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tPUBLISHER\tDESCRIPTION")
		for _, s := range results {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Version, s.PublisherID, s.Description)
		}
		w.Flush()

		return nil
	},
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <skill-name>",
	Short: "Install a skill from the marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		skillName := args[0]
		client := marketplace.NewClient()

		skills, err := client.Search(context.Background(), skillName)
		if err != nil {
			return fmt.Errorf("searching marketplace: %w", err)
		}

		var found *marketplace.MarketSkill
		for i := range skills {
			if skills[i].Name == skillName {
				found = &skills[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("skill not found in marketplace: %s", skillName)
		}

		repoURL := found.RepoURL
		if repoURL == "" {
			return fmt.Errorf("no repository URL for skill %s", skillName)
		}

		ghClient := github.NewClient(cfg.GitHubToken)
		importer := github.NewImporter(ghClient, cfg.SkillsPath)

		fmt.Printf("Installing %s from %s...\n", skillName, repoURL)
		results, err := importer.ImportFromURL(context.Background(), repoURL, found.Path)
		if err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		for _, r := range results {
			if !r.Skipped {
				fmt.Printf("  Installed %s → %s\n", r.SkillName, r.Path)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(marketplaceCmd)
	marketplaceCmd.AddCommand(marketplaceBrowseCmd)
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
}
