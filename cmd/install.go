package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
	"github.com/ying-sun1/skill-tui/internal/tui/components"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

var installCmd = &cobra.Command{
	Use:   "install <skill-name>",
	Short: "Install a skill to selected platform(s)",
	Long: `Install a skill from the central registry to one or more platforms.

Without flags, shows an interactive platform selector.
Use --platform to target a specific platform, or --all for all detected platforms.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		registry := skill.NewRegistry(cfg.SkillsPath)
		if err := registry.EnsureDir(); err != nil {
			return err
		}

		skillName := args[0]
		platformFlag, _ := cmd.Flags().GetString("platform")
		installAll, _ := cmd.Flags().GetBool("all")

		targetSkill, err := registry.GetSkill(skillName)
		if err != nil {
			return fmt.Errorf("skill not found in registry: %s\nAdd it first by copying to %s", skillName, cfg.SkillsPath)
		}

		// --platform: single target
		if platformFlag != "" {
			p := platform.FindPlatform(cfg, platformFlag)
			if p == nil {
				return fmt.Errorf("platform not found: %s", platformFlag)
			}
			if err := platform.Install(p.SkillsDir, targetSkill.Path, targetSkill.Name); err != nil {
				return err
			}
			fmt.Printf("Installed %s → %s\n", targetSkill.Name, p.Name)
			return nil
		}

		// --all: every detected platform
		if installAll {
			return installToAll(targetSkill, cfg)
		}

		// No flag: interactive selector
		return runInstallSelector(targetSkill, cfg)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringP("platform", "p", "", "Target platform name")
	installCmd.Flags().Bool("all", false, "Install to all detected platforms")
}

func installToAll(s *skill.Skill, cfg *config.Config) error {
	platforms := platform.DetectInstalled(cfg)
	count := 0
	for _, p := range platforms {
		if p.Category == "central" || !p.Installed {
			continue
		}
		if err := platform.Install(p.SkillsDir, s.Path, s.Name); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to install to %s: %v\n", p.Name, err)
			continue
		}
		fmt.Printf("Installed %s → %s\n", s.Name, p.Name)
		count++
	}
	fmt.Printf("\nDone: installed to %d platform(s)\n", count)
	return nil
}

type installSelectModel struct {
	theme     styles.Theme
	skill     *skill.Skill
	platforms []platform.Platform
	platMap   map[string]string
	multiSel  components.MultiSelectModel
	registry  *skill.Registry
}

func runInstallSelector(s *skill.Skill, cfg *config.Config) error {
	theme := styles.NewTheme(cfg.Theme)
	platforms := platform.DetectInstalled(cfg)
	platMap := make(map[string]string)
	for _, p := range platforms {
		if p.Category != "central" {
			platMap[p.Name] = p.SkillsDir
		}
	}

	linked := platform.LinkedPlatforms(platMap, s.Name)

	var items []components.MultiSelectItem
	for _, p := range platforms {
		if p.Category == "central" || !p.Installed {
			continue
		}
		isLinked := false
		for _, l := range linked {
			if l == p.Name {
				isLinked = true
				break
			}
		}
		label := p.Name
		if isLinked {
			label += " (installed)"
		}
		items = append(items, components.MultiSelectItem{
			Key:   p.Name,
			Label: label,
			Desc:  p.SkillsDir,
		})
	}

	if len(items) == 0 {
		fmt.Println("No installed platforms detected.")
		return nil
	}

	model := installSelectModel{
		theme:     theme,
		skill:     s,
		platforms: platforms,
		platMap:   platMap,
		registry:  skill.NewRegistry(cfg.SkillsPath),
	}
	model.multiSel = components.NewMultiSelect(theme, fmt.Sprintf("Install %s to:", s.Name), items)

	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

func (m installSelectModel) Init() tea.Cmd {
	return nil
}

func (m installSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.multiSel.Selected()
			if len(selected) == 0 {
				return m, tea.Quit
			}
			count := 0
			for _, name := range selected {
				dir := m.platMap[name]
				if err := platform.Install(dir, m.skill.Path, m.skill.Name); err != nil {
					continue
				}
				count++
			}
			fmt.Printf("\nInstalled %s to %d platform(s)\n", m.skill.Name, count)
			return m, tea.Quit
		case "esc", "q":
			return m, tea.Quit
		}

		if msg.String() == " " || msg.String() == "a" ||
			key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))) ||
			key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))) {
			var cmd tea.Cmd
			m.multiSel, cmd = m.multiSel.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m installSelectModel) View() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render(fmt.Sprintf("Install: %s", m.skill.Name)))
	b.WriteString("\n\n")
	b.WriteString(m.multiSel.View())
	return b.String()
}
