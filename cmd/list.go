package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	Long:  `List all skills in the central registry. Use --platform to filter by platform.`,
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

		if len(skills) == 0 {
			fmt.Println("No skills found in", cfg.SkillsPath)
			return nil
		}

		platformName, _ := cmd.Flags().GetString("platform")
		verbose, _ := cmd.Flags().GetBool("verbose")

		platformMap := buildPlatformMap(cfg)

		if platformName != "" {
			p := platform.FindPlatform(cfg, platformName)
			if p == nil {
				return fmt.Errorf("platform not found: %s", platformName)
			}
			skills = filterByPlatform(skills, p.SkillsDir)
		}

		if verbose {
			printVerboseTable(skills, platformMap)
		} else {
			printSimpleTable(skills, platformMap)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("platform", "p", "", "Filter by platform name")
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
}

func buildPlatformMap(cfg *config.Config) map[string]string {
	m := make(map[string]string)
	for _, p := range cfg.Platforms {
		if p.Category != "central" {
			m[p.Name] = p.SkillsDir
		}
	}
	return m
}

func filterByPlatform(skills []skill.Skill, platformDir string) []skill.Skill {
	var filtered []skill.Skill
	for _, s := range skills {
		if platform.IsLinked(platformDir, s.Name) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func printSimpleTable(skills []skill.Skill, platformMap map[string]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
	for _, s := range skills {
		desc := s.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Version, desc)
	}
	w.Flush()
}

func printVerboseTable(skills []skill.Skill, platformMap map[string]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tAUTHOR\tPLATFORMS")
	for _, s := range skills {
		platforms := platform.LinkedPlatforms(platformMap, s.Name)
		platStr := "-"
		if len(platforms) > 0 {
			platStr = fmt.Sprintf("%d: %v", len(platforms), platforms)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Version, s.Author, platStr)
	}
	w.Flush()
}
