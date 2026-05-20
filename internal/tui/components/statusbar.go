package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type StatusBar struct {
	Theme          styles.Theme
	Width          int
	SkillCount     int
	Platforms      int
	Path           string
	Tab            string
	PluginInfo     string
	Message        string
	MessageIsError bool
}

func (s StatusBar) View() string {
	// If there's a status message, show it prominently
	if s.Message != "" {
		left := s.Theme.StatusAccent.Render(fmt.Sprintf(" %s ", s.Tab))
		var msgStyle lipgloss.Style
		if s.MessageIsError {
			msgStyle = s.Theme.StatusBar.
				Background(lipgloss.Color("#e64553")).
				Foreground(lipgloss.Color("#ffffff"))
		} else {
			msgStyle = s.Theme.StatusBar.
				Background(lipgloss.Color("#40a02b")).
				Foreground(lipgloss.Color("#ffffff"))
		}
		msg := msgStyle.Render(fmt.Sprintf(" %s ", s.Message))

		fillWidth := s.Width - lipgloss.Width(left) - lipgloss.Width(msg)
		if fillWidth < 0 {
			fillWidth = 0
		}
		fill := s.Theme.StatusBar.Render(string(make([]byte, fillWidth)))

		return lipgloss.JoinHorizontal(lipgloss.Top,
			s.Theme.StatusBar.Render(left),
			msg,
			fill,
		)
	}

	left := s.Theme.StatusAccent.Render(fmt.Sprintf(" %s ", s.Tab))

	var center string
	if s.PluginInfo != "" {
		center = s.Theme.StatusText.Render(fmt.Sprintf(" %s ", s.PluginInfo))
	} else {
		center = s.Theme.StatusText.Render(fmt.Sprintf(" %d skills · %d platforms ", s.SkillCount, s.Platforms))
	}
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
