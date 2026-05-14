package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
)

var removeCmd = &cobra.Command{
	Use:   "remove <skill-name>",
	Short: "Remove a skill from platform(s)",
	Long: `Remove a skill from one or more platforms.

Use --platform to remove from a specific platform.
Use --purge to also remove from the central registry.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		skillName := args[0]
		platformName, _ := cmd.Flags().GetString("platform")
		purge, _ := cmd.Flags().GetBool("purge")
		force, _ := cmd.Flags().GetBool("force")

		registry := skill.NewRegistry(cfg.SkillsPath)
		platformMap := buildPlatformMap(cfg)

		if platformName != "" {
			p := platform.FindPlatform(cfg, platformName)
			if p == nil {
				return fmt.Errorf("platform not found: %s", platformName)
			}
			if err := platform.Uninstall(p.SkillsDir, skillName); err != nil {
				return err
			}
			fmt.Printf("Removed %s from %s\n", skillName, p.Name)

			if purge {
				if err := registry.RemoveSkill(skillName); err != nil {
					return err
				}
				fmt.Printf("Purged %s from central registry\n", skillName)
			}
			return nil
		}

		if !force {
			fmt.Printf("Remove %s from all platforms? [y/N] ", skillName)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		for name, dir := range platformMap {
			if platform.IsLinked(dir, skillName) {
				platform.Uninstall(dir, skillName)
				fmt.Printf("Removed %s from %s\n", skillName, name)
			}
		}

		if purge {
			if err := registry.RemoveSkill(skillName); err != nil {
				return err
			}
			fmt.Printf("Purged %s from central registry\n", skillName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("platform", "p", "", "Remove from specific platform")
	removeCmd.Flags().Bool("purge", false, "Also remove from central registry")
	removeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
