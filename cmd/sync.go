package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync skills between central registry and platforms",
	Long: `Synchronize skills from the central registry to all platform directories.

Detects and fixes broken symlinks, creates missing links, and removes
stale links that point to deleted skills.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		registry := skill.NewRegistry(cfg.SkillsPath)
		skills, err := registry.ListSkills()
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		platformName, _ := cmd.Flags().GetString("platform")

		var platforms []platform.Platform
		if platformName != "" {
			p := platform.FindPlatform(cfg, platformName)
			if p == nil {
				return fmt.Errorf("platform not found: %s", platformName)
			}
			platforms = []platform.Platform{*p}
		} else {
			platforms = platform.DetectInstalled(cfg)
		}

		skillNames := make(map[string]string)
		for _, s := range skills {
			skillNames[s.Name] = s.Path
		}

		for _, p := range platforms {
			if p.Category == "central" {
				continue
			}

			fmt.Printf("Syncing %s...\n", p.Name)

			for _, s := range skills {
				if platform.IsLinked(p.SkillsDir, s.Name) {
					continue
				}

				if dryRun {
					fmt.Printf("  [dry-run] Would link %s → %s\n", s.Name, p.Name)
					continue
				}

				if err := platform.Install(p.SkillsDir, s.Path, s.Name); err != nil {
					fmt.Printf("  Error linking %s: %v\n", s.Name, err)
					continue
				}
				fmt.Printf("  Linked %s → %s\n", s.Name, p.Name)
			}

			broken, err := platform.BrokenLinks(p.SkillsDir)
			if err != nil {
				continue
			}
			for _, b := range broken {
				if dryRun {
					fmt.Printf("  [dry-run] Would remove broken link: %s\n", b)
					continue
				}
				platform.Uninstall(p.SkillsDir, b)
				if _, ok := skillNames[b]; ok {
					platform.Install(p.SkillsDir, skillNames[b], b)
					fmt.Printf("  Fixed broken link: %s\n", b)
				} else {
					fmt.Printf("  Removed stale link: %s\n", b)
				}
			}
		}

		fmt.Println("Sync complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().Bool("dry-run", false, "Preview changes without making them")
	syncCmd.Flags().StringP("platform", "p", "", "Sync only a specific platform")
}
