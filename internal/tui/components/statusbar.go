package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type StatusBar struct {
	Theme      styles.Theme
	Width      int
	SkillCount int
	Platforms  int
	Path       string
	Tab        string
}

func (s StatusBar) View() string {
	left := s.Theme.StatusAccent.Render(fmt.Sprintf(" %s ", s.Tab))
	center := s.Theme.StatusText.Render(fmt.Sprintf(" %d skills · %d platforms ", s.SkillCount, s.Platforms))
	right := s.Theme.StatusText.Render(fmt.Sprintf(" %s ", s.Path))

	fillWidth := s.Width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
	if fillWidth < 0 {
		fillWidth = 0
	}
	fill := s.Theme.StatusBar.Render(string(make([]byte, fillWidth)))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		s.Theme.StatusBar.Render(left),
		s.Theme.StatusBar.Render(center),
		fill,
		s.Theme.StatusBar.Render(right),
	)

	return bar
}
