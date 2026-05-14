package cmd

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/ying-sun1/skill-tui/internal/config"
	tui "github.com/ying-sun1/skill-tui/internal/tui"
)

func runTUI(cfg *config.Config) error {
	app := tui.NewApp(cfg)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
