package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/skill"
	"github.com/ying-sun1/skill-tui/internal/tui/components"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type tab int

const (
	tabSkills tab = iota
	tabMarketplace
	tabCollections
	tabSettings
)

var tabNames = []string{"Skills", "Marketplace", "Collections", "Settings"}

type view int

const (
	viewList view = iota
	viewDetail
	viewPlatformSelect
	viewDetailPlatformSelect
)

type AppModel struct {
	cfg      *config.Config
	theme    styles.Theme
	registry *skill.Registry

	tabs        []string
	activeTab   tab
	currentView view

	skills   []skill.Skill
	selected map[string]bool
	cursor   int
	scroll   int

	platforms   []platform.Platform
	platformMap map[string]string

	detailSkill *skill.Skill
	fullContent bool

	search    components.SearchModel
	statusBar components.StatusBar
	multiSel  components.MultiSelectModel

	width  int
	height int

	settingsCursor  int
	settingsEditing bool
	settingsInput   textinput.Model

	lastClickTime time.Time
	lastClickRow  int

	err error
}

const (
	settingTheme = iota
	settingAccent
	settingSkillsPath
	settingsCount
)

var themeOptions = []string{"mocha", "latte"}

func NewApp(cfg *config.Config) AppModel {
	theme := styles.NewThemeWithAccent(cfg.Theme, cfg.AccentColor)
	registry := skill.NewRegistry(cfg.SkillsPath)

	platforms := platform.DetectInstalled(cfg)
	platformMap := make(map[string]string)
	for _, p := range platforms {
		if p.Category != "central" {
			platformMap[p.Name] = p.SkillsDir
		}
	}

	search := components.NewSearch(theme)

	app := AppModel{
		cfg:         cfg,
		theme:       theme,
		registry:    registry,
		tabs:        tabNames,
		selected:    make(map[string]bool),
		platforms:   platforms,
		platformMap: platformMap,
		search:      search,
	}
	app.loadSkills()
	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.Width = msg.Width

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Tab bar click (Y=0)
			if msg.Y == 0 {
				x := msg.X
				for i, name := range m.tabs {
					style := m.theme.InactiveTab
					if tab(i) == m.activeTab {
						style = m.theme.ActiveTab
					}
					rendered := style.Render(fmt.Sprintf(" %d %s ", i+1, name))
					w := lipgloss.Width(rendered)
					if x < w {
						m.activeTab = tab(i)
						m.currentView = viewList
						m.settingsCursor = 0
						break
					}
					x -= w
				}
			}

			// Skill list click
			if m.activeTab == tabSkills && m.currentView == viewList {
				const skillsStartY = 6
				visIdx := msg.Y - skillsStartY
				idx := visIdx + m.scroll
				filtered := m.filteredSkills()
				if visIdx >= 0 && idx >= 0 && idx < len(filtered) {
					now := time.Now()
					if idx == m.lastClickRow && now.Sub(m.lastClickTime) < 400*time.Millisecond {
						// Double-click: toggle selection
						name := filtered[idx].Name
						m.selected[name] = !m.selected[name]
						m.lastClickTime = time.Time{}
					} else {
						// Single click: move cursor
						m.cursor = idx
						m.lastClickTime = now
						m.lastClickRow = idx
					}
				}
			}
		}

	case tea.KeyMsg:
		if m.search.Active() {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if msg.String() == "esc" {
				m.search.Blur()
				m.search.Reset()
				return m, nil
			}
			if msg.String() == "enter" {
				m.search.Blur()
				return m, nil
			}
			m.filterSkills()
			return m, cmd
		}

		if m.currentView == viewPlatformSelect {
			return m.handlePlatformSelect(msg, false)
		}

		if m.currentView == viewDetailPlatformSelect {
			return m.handlePlatformSelect(msg, true)
		}

		if m.currentView == viewDetail {
			return m.handleDetail(msg)
		}

		if m.activeTab == tabSettings {
			return m.handleSettings(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1", "2", "3", "4":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.tabs) {
				m.activeTab = tab(idx)
				m.currentView = viewList
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % tab(len(m.tabs))
			m.currentView = viewList
		case "/":
			m.search.Focus()
			return m, nil
		case "up", "k":
			filtered := m.filteredSkills()
			if m.cursor > 0 {
				m.cursor--
			}
			m.adjustScroll(filtered)
		case "down", "j":
			filtered := m.filteredSkills()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}
			m.adjustScroll(filtered)
		case " ":
			if len(m.skills) > 0 {
				name := m.skills[m.cursor].Name
				m.selected[name] = !m.selected[name]
			}
		case "a":
			allSelected := len(m.selected) == len(m.skills)
			m.selected = make(map[string]bool)
			if !allSelected {
				for _, s := range m.skills {
					m.selected[s.Name] = true
				}
			}
		case "enter", "d":
			if len(m.skills) > 0 {
				s := m.skills[m.cursor]
				m.detailSkill = &s
				m.currentView = viewDetail
			}
		case "p":
			if len(m.selected) > 0 {
				m.showPlatformSelect()
			}
		case "x":
			m.removeSelected()
		case "r":
			m.loadSkills()
		case "o":
			filtered := m.filteredSkills()
			if m.cursor >= 0 && m.cursor < len(filtered) {
				dir := filtered[m.cursor].Path
				opener := "open"
				if runtime.GOOS == "linux" {
					opener = "xdg-open"
				}
				exec.Command(opener, dir).Start()
			}
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	tabBar := m.renderTabBar()
	content := m.renderContent()
	status := m.renderStatusBar()

	availHeight := m.height - lipgloss.Height(tabBar) - lipgloss.Height(status) - 1
	if availHeight < 1 {
		availHeight = 1
	}

	contentLines := strings.Split(content, "\n")
	if len(contentLines) > availHeight {
		contentLines = contentLines[:availHeight]
	}
	content = strings.Join(contentLines, "\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		content,
		status,
	)
}

func (m AppModel) renderTabBar() string {
	var tabs []string
	for i, name := range m.tabs {
		style := m.theme.InactiveTab
		if tab(i) == m.activeTab {
			style = m.theme.ActiveTab
		}
		tabs = append(tabs, style.Render(fmt.Sprintf(" %d %s ", i+1, name)))
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	return tabRow
}

func (m AppModel) renderContent() string {
	switch m.activeTab {
	case tabSkills:
		return m.renderSkillsTab()
	case tabMarketplace:
		return m.theme.Dimmed.Render("\n  Marketplace — Coming soon...")
	case tabCollections:
		return m.theme.Dimmed.Render("\n  Collections — Coming soon...")
	case tabSettings:
		return m.renderSettingsTab()
	default:
		return ""
	}
}

func (m AppModel) renderSkillsTab() string {
	switch m.currentView {
	case viewDetail:
		return m.renderDetail()
	case viewPlatformSelect, viewDetailPlatformSelect:
		return m.multiSel.View()
	default:
		return m.renderSkillList()
	}
}

func (m AppModel) renderSkillList() string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Skills"))
	b.WriteString("  ")
	b.WriteString(m.search.View())
	b.WriteString("\n\n")

	filtered := m.filteredSkills()
	if len(filtered) == 0 {
		if m.search.Value() != "" {
			b.WriteString(m.theme.Dimmed.Render("  No matching skills found"))
		} else {
			b.WriteString(m.theme.Dimmed.Render("  No skills found in " + m.cfg.SkillsPath))
			b.WriteString("\n")
			b.WriteString(m.theme.Dimmed.Render("  Add skills to ~/.agents/skills/ to get started"))
		}
		return b.String()
	}

	// Collect installed platform names for column headers
	var platCols []string
	for _, p := range m.platforms {
		if p.Category != "central" && p.Installed {
			platCols = append(platCols, p.Name)
		}
	}
	sort.Strings(platCols)

	const nameWidth = 24
	const colWidth = 8

	// Header row
	header := fmt.Sprintf("     %-*s", nameWidth, "")
	for _, p := range platCols {
		short := abbreviatePlatform(p)
		header += fmt.Sprintf("%-*s", colWidth, short)
	}
	b.WriteString(m.theme.Subtitle.Render(header))
	b.WriteString("\n")

	// Separator
	sepLen := 5 + nameWidth + colWidth*len(platCols)
	b.WriteString(m.theme.Dimmed.Render("  " + strings.Repeat("─", sepLen-2)))
	b.WriteString("\n")

	// Skill rows (windowed by scroll)
	visible := m.visibleSkillRows()
	start := m.scroll
	end := start + visible
	if end > len(filtered) {
		end = len(filtered)
	}
	for i := start; i < end; i++ {
		s := filtered[i]
		cursor := " "
		if i == m.cursor {
			cursor = m.theme.Cursor.Render(">")
		}

		check := m.theme.CheckboxOff
		if m.selected[s.Name] {
			check = m.theme.CheckboxOn
		}

		// Truncate skill name to fit column
		displayName := s.Name
		if len(displayName) > nameWidth-1 {
			displayName = displayName[:nameWidth-4] + "..."
		}

		var nameStyled string
		if i == m.cursor {
			nameStyled = m.theme.Selected.Render(fmt.Sprintf("%-*s", nameWidth, displayName))
		} else {
			nameStyled = m.theme.Normal.Render(fmt.Sprintf("%-*s", nameWidth, displayName))
		}

		// Platform columns with ✓/·
		var cols strings.Builder
		for _, p := range platCols {
			dir := m.platformMap[p]
			if platform.IsLinked(dir, s.Name) {
				cols.WriteString(m.theme.Success.Render(fmt.Sprintf("%-*s", colWidth, "✓")))
			} else {
				cols.WriteString(m.theme.Dimmed.Render(fmt.Sprintf("%-*s", colWidth, "·")))
			}
		}

		// Description (truncated)
		desc := ""
		if s.Description != "" {
			d := s.Description
			if len(d) > 35 {
				d = d[:32] + "..."
			}
			desc = m.theme.Dimmed.Render(d)
		}

		b.WriteString(fmt.Sprintf(" %s %s %s%s %s\n", cursor, check, nameStyled, cols.String(), desc))
	}

	// Scroll indicator
	if len(filtered) > visible {
		b.WriteString(m.theme.Dimmed.Render(fmt.Sprintf("  ─── %d-%d / %d ───", start+1, end, len(filtered))))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString(m.theme.Dimmed.Render("  ↑/k↓/j: navigate  Space: select  a: all  Enter/d: detail  o: open  p: install  x: remove  /: search  r: refresh"))

	return b.String()
}

func (m AppModel) renderDetail() string {
	if m.detailSkill == nil {
		return ""
	}
	s := m.detailSkill
	var b strings.Builder

	b.WriteString(m.theme.Title.Render(s.Name))
	b.WriteString("\n")

	if s.Version != "" {
		b.WriteString(m.theme.Accent.Render("Version: "))
		b.WriteString(m.theme.Normal.Render(s.Version))
		b.WriteString("  ")
	}
	if s.Author != "" {
		b.WriteString(m.theme.Accent.Render("Author: "))
		b.WriteString(m.theme.Normal.Render(s.Author))
		b.WriteString("  ")
	}
	if s.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.theme.Normal.Render(s.Description))
	}
	b.WriteString("\n\n")

	b.WriteString(m.theme.Subtitle.Render("Installed Platforms"))
	b.WriteString("\n")
	linked := platform.LinkedPlatforms(m.platformMap, s.Name)
	if len(linked) > 0 {
		for _, p := range linked {
			b.WriteString(m.theme.Success.Render("  ✓ "))
			b.WriteString(m.theme.Normal.Render(p))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(m.theme.Dimmed.Render("  Not installed to any platform"))
		b.WriteString("\n")
	}

	if len(m.platforms) > 0 {
		availCount := len(m.platforms) - len(linked) - countByCategory(m.platforms, "central")
		if availCount > 0 {
			b.WriteString(m.theme.Dimmed.Render(fmt.Sprintf("\n  Press 'i' to select platforms for installation (%d available)", availCount)))
		}
	}

	if s.Content != "" {
		b.WriteString("\n")
		b.WriteString(m.theme.Subtitle.Render("Content"))
		b.WriteString("\n")
		content := s.Content
		if !m.fullContent {
			maxLines := 20
			lines := strings.Split(content, "\n")
			if len(lines) > maxLines {
				lines = append(lines[:maxLines], "")
				lines = append(lines, m.theme.Dimmed.Render("  ... press 'o' to expand full content"))
			}
			content = strings.Join(lines, "\n")
		}
		b.WriteString(m.theme.Normal.Render(content))
	}

	b.WriteString("\n\n")
	if m.fullContent {
		b.WriteString(m.theme.Dimmed.Render("  Esc: back  i: install to...  u: uninstall all  o: collapse content"))
	} else {
		b.WriteString(m.theme.Dimmed.Render("  Esc: back  i: install to...  u: uninstall all  o: expand content"))
	}

	return b.String()
}

func abbreviatePlatform(name string) string {
	abbr := map[string]string{
		"claude-code": "claude",
		"codex-cli":   "codex",
		"gemini-cli":  "gemini",
	}
	if short, ok := abbr[name]; ok {
		return short
	}
	if len(name) > 8 {
		return name[:7] + "."
	}
	return name
}

func (m AppModel) renderSettingsTab() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Settings"))
	b.WriteString("\n\n")

	type settingItem struct {
		label string
		value string
	}

	items := []settingItem{
		{"Theme", m.cfg.Theme},
		{"Accent Color", m.cfg.AccentColor},
		{"Skills Path", m.cfg.SkillsPath},
	}

	for i, item := range items {
		cursor := "  "
		if i == m.settingsCursor {
			cursor = m.theme.Cursor.Render("> ")
		}

		label := m.theme.Accent.Render(fmt.Sprintf("%-14s", item.label))

		if m.settingsEditing && i == m.settingsCursor && i == settingSkillsPath {
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, label, m.settingsInput.View()))
		} else {
			valStyle := m.theme.Normal
			if i == m.settingsCursor {
				valStyle = m.theme.Selected
			}
			hint := ""
			if i == m.settingsCursor {
				switch i {
				case settingTheme, settingAccent:
					hint = m.theme.Dimmed.Render("  ← Enter to cycle")
				case settingSkillsPath:
					hint = m.theme.Dimmed.Render("  ← Enter to edit")
				}
			}
			b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, label, valStyle.Render(item.value), hint))
		}
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Subtitle.Render("Detected Platforms"))
	b.WriteString("\n")
	for _, p := range m.platforms {
		if p.Category == "central" {
			continue
		}
		icon := m.theme.Success.Render("✓")
		if !p.Installed {
			icon = m.theme.Dimmed.Render("○")
		}
		b.WriteString(fmt.Sprintf("  %s %-15s %s\n", icon, p.Name, m.theme.Dimmed.Render(p.SkillsDir)))
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Dimmed.Render("  ↑/k↓/j: navigate  Enter: edit  q: quit"))

	return b.String()
}

func (m AppModel) handleSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsEditing {
		switch msg.String() {
		case "enter":
			m.cfg.SkillsPath = m.settingsInput.Value()
			m.settingsEditing = false
			m.settingsInput.Blur()
			m.applyAndSaveSettings()
		case "esc":
			m.settingsEditing = false
			m.settingsInput.Blur()
		default:
			var cmd tea.Cmd
			m.settingsInput, cmd = m.settingsInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1", "2", "3", "4":
		idx := int(msg.String()[0] - '1')
		if idx < len(m.tabs) {
			m.activeTab = tab(idx)
			m.currentView = viewList
		}
	case "tab":
		m.activeTab = (m.activeTab + 1) % tab(len(m.tabs))
		m.currentView = viewList
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < settingsCount-1 {
			m.settingsCursor++
		}
	case "enter":
		switch m.settingsCursor {
		case settingTheme:
			m.cfg.Theme = cycleOption(themeOptions, m.cfg.Theme)
			m.applyAndSaveSettings()
		case settingAccent:
			m.cfg.AccentColor = cycleOption(styles.AccentColors, m.cfg.AccentColor)
			m.applyAndSaveSettings()
		case settingSkillsPath:
			ti := textinput.New()
			ti.SetValue(m.cfg.SkillsPath)
			ti.Focus()
			ti.CharLimit = 200
			ti.Width = 60
			ti.PromptStyle = m.theme.Accent
			ti.TextStyle = m.theme.Normal
			m.settingsInput = ti
			m.settingsEditing = true
		}
	}
	return m, nil
}

func (m *AppModel) applyAndSaveSettings() {
	m.theme = styles.NewThemeWithAccent(m.cfg.Theme, m.cfg.AccentColor)
	m.search = components.NewSearch(m.theme)
	_ = config.Save(m.cfg)
}

func cycleOption(options []string, current string) string {
	for i, o := range options {
		if o == current {
			return options[(i+1)%len(options)]
		}
	}
	return options[0]
}

func (m AppModel) renderStatusBar() string {
	m.statusBar = components.StatusBar{
		Theme:      m.theme,
		Width:      m.width,
		SkillCount: len(m.skills),
		Platforms:  len(m.platforms),
		Path:       m.cfg.SkillsPath,
		Tab:        m.tabs[m.activeTab],
	}
	return m.statusBar.View()
}

func (m *AppModel) loadSkills() {
	skills, err := m.registry.ListSkills()
	if err != nil {
		m.err = err
		return
	}
	m.skills = skills
	m.selected = make(map[string]bool)
	m.cursor = 0
	m.scroll = 0
}

func (m *AppModel) filterSkills() {
	m.cursor = 0
	m.scroll = 0
}

func (m AppModel) filteredSkills() []skill.Skill {
	q := m.search.Value()
	if q == "" {
		return m.skills
	}

	var filtered []skill.Skill
	q = strings.ToLower(q)
	for _, s := range m.skills {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Description), q) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// visibleSkillRows returns how many skill rows fit in the viewport.
// Layout: TabBar(1) + Title+Search(1) + blank(1) + header(1) + separator(1) + helpbar(2) + statusbar(1) = 8 overhead
func (m AppModel) visibleSkillRows() int {
	overhead := 8
	rows := m.height - overhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m *AppModel) adjustScroll(filtered []skill.Skill) {
	visible := m.visibleSkillRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	maxScroll := len(filtered) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *AppModel) showPlatformSelect() {
	var items []components.MultiSelectItem
	for _, p := range m.platforms {
		if p.Category == "central" || !p.Installed {
			continue
		}
		items = append(items, components.MultiSelectItem{
			Key:   p.Name,
			Label: p.Name,
			Desc:  p.SkillsDir,
		})
	}
	if len(items) == 0 {
		return
	}
	m.multiSel = components.NewMultiSelect(m.theme, "Select target platforms", items)
	m.currentView = viewPlatformSelect
}

func (m *AppModel) showDetailPlatformSelect() {
	if m.detailSkill == nil {
		return
	}
	linked := platform.LinkedPlatforms(m.platformMap, m.detailSkill.Name)
	var items []components.MultiSelectItem
	for _, p := range m.platforms {
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
		suffix := ""
		if isLinked {
			suffix = " (installed)"
		}
		items = append(items, components.MultiSelectItem{
			Key:   p.Name,
			Label: p.Name + suffix,
			Desc:  p.SkillsDir,
		})
	}
	if len(items) == 0 {
		return
	}
	m.multiSel = components.NewMultiSelect(m.theme, fmt.Sprintf("Install %s to:", m.detailSkill.Name), items)
	m.currentView = viewDetailPlatformSelect
}

func (m AppModel) handlePlatformSelect(msg tea.KeyMsg, fromDetail bool) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.multiSel, cmd = m.multiSel.Update(msg)

	if msg.String() == "enter" {
		selected := m.multiSel.Selected()
		if fromDetail && m.detailSkill != nil {
			for _, name := range selected {
				dir := m.platformMap[name]
				platform.Install(dir, m.detailSkill.Path, m.detailSkill.Name)
			}
			s, _ := m.registry.GetSkill(m.detailSkill.Name)
			if s != nil {
				m.detailSkill = s
			}
			m.currentView = viewDetail
		} else {
			for _, name := range selected {
				dir := m.platformMap[name]
				for skillName := range m.selected {
					s, err := m.registry.GetSkill(skillName)
					if err != nil {
						continue
					}
					platform.Install(dir, s.Path, s.Name)
				}
			}
			m.currentView = viewList
			m.loadSkills()
		}
		return m, nil
	}

	if msg.String() == "esc" {
		if fromDetail {
			m.currentView = viewDetail
		} else {
			m.currentView = viewList
		}
		return m, nil
	}

	return m, cmd
}

func (m AppModel) handleDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.currentView = viewList
		m.detailSkill = nil
		m.fullContent = false
	case "i":
		if m.detailSkill != nil {
			m.showDetailPlatformSelect()
		}
	case "u":
		if m.detailSkill != nil {
			for _, p := range m.platforms {
				if p.Category == "central" {
					continue
				}
				platform.Uninstall(p.SkillsDir, m.detailSkill.Name)
			}
			s, _ := m.registry.GetSkill(m.detailSkill.Name)
			if s != nil {
				m.detailSkill = s
			}
		}
	case "o":
		m.fullContent = !m.fullContent
	}
	return m, nil
}

func (m *AppModel) removeSelected() {
	for name := range m.selected {
		for _, p := range m.platforms {
			if p.Category == "central" {
				continue
			}
			platform.Uninstall(p.SkillsDir, name)
		}
	}
	m.loadSkills()
}

func countByCategory(platforms []platform.Platform, cat string) int {
	count := 0
	for _, p := range platforms {
		if p.Category == cat {
			count++
		}
	}
	return count
}
