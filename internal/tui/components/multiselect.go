package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbletea"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type MultiSelectItem struct {
	Key   string
	Label string
	Desc  string
}

type MultiSelectModel struct {
	theme    styles.Theme
	items    []MultiSelectItem
	selected map[string]bool
	cursor   int
	width    int
	height   int
	title    string
}

func NewMultiSelect(theme styles.Theme, title string, items []MultiSelectItem) MultiSelectModel {
	return MultiSelectModel{
		theme:    theme,
		items:    items,
		selected: make(map[string]bool),
		title:    title,
	}
}

func (m MultiSelectModel) Init() tea.Cmd {
	return nil
}

func (m MultiSelectModel) Update(msg tea.Msg) (MultiSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			k := m.items[m.cursor].Key
			m.selected[k] = !m.selected[k]
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			allSelected := len(m.selected) == len(m.items)
			m.selected = make(map[string]bool)
			if !allSelected {
				for _, item := range m.items {
					m.selected[item.Key] = true
				}
			}
		}
	}
	return m, nil
}

func (m MultiSelectModel) View() string {
	var b strings.Builder

	if m.title != "" {
		b.WriteString(m.theme.Subtitle.Render(m.title))
		b.WriteString("\n\n")
	}

	for i, item := range m.items {
		cursor := " "
		if i == m.cursor {
			cursor = m.theme.Cursor.Render(">")
		}

		check := m.theme.CheckboxOff
		if m.selected[item.Key] {
			check = m.theme.CheckboxOn
		}

		label := m.theme.Normal.Render(item.Label)
		if i == m.cursor {
			label = m.theme.Selected.Render(item.Label)
		}

		b.WriteString(fmt.Sprintf(" %s %s %s", cursor, check, label))
		if item.Desc != "" {
			b.WriteString("  ")
			b.WriteString(m.theme.Dimmed.Render(item.Desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Dimmed.Render("Space: select  a: toggle all  Enter: confirm  Esc: cancel"))

	return b.String()
}

func (m MultiSelectModel) Selected() []string {
	var keys []string
	for _, item := range m.items {
		if m.selected[item.Key] {
			keys = append(keys, item.Key)
		}
	}
	return keys
}

func (m MultiSelectModel) SetSize(width, height int) MultiSelectModel {
	m.width = width
	m.height = height
	return m
}
