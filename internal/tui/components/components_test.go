package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

func testTheme() styles.Theme {
	return styles.NewTheme("mocha")
}

// --- StatusBar tests ---

func TestStatusBar_View_Default(t *testing.T) {
	sb := StatusBar{
		Theme:      testTheme(),
		Width:      80,
		SkillCount: 5,
		Platforms:  3,
		Path:       "/test/path",
		Tab:        "Skills",
	}

	view := sb.View()
	if view == "" {
		t.Error("View should not be empty")
	}
	if !strings.Contains(view, "Skills") {
		t.Error("should contain tab name")
	}
}

func TestStatusBar_View_WithMessage(t *testing.T) {
	sb := StatusBar{
		Theme:          testTheme(),
		Width:          80,
		Tab:            "Plugins",
		Message:        "Install succeeded!",
		MessageIsError: false,
	}

	view := sb.View()
	if !strings.Contains(view, "Install succeeded!") {
		t.Error("should contain success message")
	}
}

func TestStatusBar_View_WithErrorMessage(t *testing.T) {
	sb := StatusBar{
		Theme:          testTheme(),
		Width:          80,
		Tab:            "Plugins",
		Message:        "Clone failed",
		MessageIsError: true,
	}

	view := sb.View()
	if !strings.Contains(view, "Clone failed") {
		t.Error("should contain error message")
	}
}

func TestStatusBar_View_WithPluginInfo(t *testing.T) {
	sb := StatusBar{
		Theme:      testTheme(),
		Width:      80,
		Tab:        "Plugins",
		PluginInfo: "3 plugins installed",
	}

	view := sb.View()
	if !strings.Contains(view, "3 plugins installed") {
		t.Error("should contain plugin info")
	}
}

func TestStatusBar_View_NarrowWidth(t *testing.T) {
	sb := StatusBar{
		Theme:   testTheme(),
		Width:   10,
		Tab:     "Skills",
		Message: "A very long message that exceeds the width",
	}

	// Should not panic
	view := sb.View()
	if view == "" {
		t.Error("View should not be empty even at narrow width")
	}
}

// --- MultiSelect tests ---

func TestMultiSelect_New(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "Item A", Desc: "desc A"},
		{Key: "b", Label: "Item B"},
	}
	m := NewMultiSelect(testTheme(), "Select items", items)

	if len(m.Selected()) != 0 {
		t.Error("should start with no selections")
	}
}

func TestMultiSelect_Init(t *testing.T) {
	m := NewMultiSelect(testTheme(), "Test", nil)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestMultiSelect_SelectItem(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
		{Key: "b", Label: "B"},
		{Key: "c", Label: "C"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Press space to select first item
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "a" {
		t.Errorf("Selected = %v, want [a]", selected)
	}
}

func TestMultiSelect_NavigateDown(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
		{Key: "b", Label: "B"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Move down and select
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "b" {
		t.Errorf("Selected = %v, want [b]", selected)
	}
}

func TestMultiSelect_NavigateUp(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
		{Key: "b", Label: "B"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Move down then up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	selected := m.Selected()
	if len(selected) != 1 || selected[0] != "a" {
		t.Errorf("Selected = %v, want [a]", selected)
	}
}

func TestMultiSelect_ToggleAll(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
		{Key: "b", Label: "B"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Select all
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(m.Selected()) != 2 {
		t.Errorf("after select all: %d, want 2", len(m.Selected()))
	}

	// Toggle all off
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(m.Selected()) != 0 {
		t.Errorf("after deselect all: %d, want 0", len(m.Selected()))
	}
}

func TestMultiSelect_View(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A", Desc: "Description"},
		{Key: "b", Label: "B"},
	}
	m := NewMultiSelect(testTheme(), "Pick items", items)

	view := m.View()
	if !strings.Contains(view, "Pick items") {
		t.Error("should contain title")
	}
	if !strings.Contains(view, "Description") {
		t.Error("should contain description")
	}
	if !strings.Contains(view, "Space: select") {
		t.Error("should contain help text")
	}
}

func TestMultiSelect_SetSize(t *testing.T) {
	m := NewMultiSelect(testTheme(), "Test", nil)
	m = m.SetSize(100, 50)
	if m.width != 100 || m.height != 50 {
		t.Errorf("size = %dx%d, want 100x50", m.width, m.height)
	}
}

func TestMultiSelect_CursorBounds(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
		{Key: "b", Label: "B"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Try moving up when at top
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 0 {
		t.Error("cursor should stay at 0")
	}

	// Move to bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Try moving past bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Errorf("cursor = %d, should stay at 1", m.cursor)
	}
}

func TestMultiSelect_DeselectItem(t *testing.T) {
	items := []MultiSelectItem{
		{Key: "a", Label: "A"},
	}
	m := NewMultiSelect(testTheme(), "Test", items)

	// Select and deselect
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if len(m.Selected()) != 0 {
		t.Error("should be deselected")
	}
}

// --- Search tests ---

func TestSearch_New(t *testing.T) {
	s := NewSearch(testTheme())
	if s.Active() {
		t.Error("should not be active initially")
	}
	if s.Value() != "" {
		t.Error("should start empty")
	}
}

func TestSearch_Init(t *testing.T) {
	s := NewSearch(testTheme())
	cmd := s.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestSearch_FocusAndBlur(t *testing.T) {
	s := NewSearch(testTheme())

	s.Focus()
	if !s.Active() {
		t.Error("should be active after Focus")
	}

	s.Blur()
	if s.Active() {
		t.Error("should not be active after Blur")
	}
}

func TestSearch_Reset(t *testing.T) {
	s := NewSearch(testTheme())
	s.Focus()
	s.Reset()

	if s.Active() {
		t.Error("should not be active after Reset")
	}
	if s.Value() != "" {
		t.Error("value should be empty after Reset")
	}
}

func TestSearch_View(t *testing.T) {
	s := NewSearch(testTheme())
	view := s.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestSearch_Update(t *testing.T) {
	s := NewSearch(testTheme())
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	// textinput updates happen through the model
	_ = s.View()
}
