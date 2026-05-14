package components

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type SearchModel struct {
	input  textinput.Model
	theme  styles.Theme
	active bool
}

func NewSearch(theme styles.Theme) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search skills..."
	ti.Prompt = "/ "
	ti.PromptStyle = theme.Accent
	ti.TextStyle = theme.Normal
	ti.CharLimit = 100
	ti.Width = 40

	return SearchModel{
		input: ti,
		theme: theme,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SearchModel) View() string {
	return m.input.View()
}

func (m SearchModel) Value() string {
	return m.input.Value()
}

func (m *SearchModel) Focus() {
	m.input.Focus()
	m.active = true
}

func (m *SearchModel) Blur() {
	m.input.Blur()
	m.active = false
}

func (m SearchModel) Active() bool {
	return m.active
}

func (m *SearchModel) Reset() {
	m.input.SetValue("")
	m.active = false
}
