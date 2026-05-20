package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/platform"
	"github.com/ying-sun1/skill-tui/internal/plugin"
	"github.com/ying-sun1/skill-tui/internal/skill"
	"github.com/ying-sun1/skill-tui/internal/tui/components"
	"github.com/ying-sun1/skill-tui/internal/tui/styles"
)

type tab int

const (
	tabSkills tab = iota
	tabMarketplace
	tabPlugin
	tabSettings
)

var tabNames = []string{"Skills", "Marketplace", "Plugin", "Settings"}

type marketplaceClonedMsg struct {
	marketplace *plugin.Marketplace
	err         error
}

type marketplaceInstalledMsg struct {
	marketplace *plugin.Marketplace
	err         error
}

type view int

const (
	viewList view = iota
	viewDetail
	viewPlatformSelect
	viewDetailPlatformSelect
	viewPluginDetail
	viewPluginInstall
	viewPluginAdd
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

	marketplaces      []plugin.Marketplace
	pluginStore       *plugin.Store
	pluginCursor      int
	pluginScroll      int
	pluginSelected    map[string]bool
	detailMarketplace *plugin.Marketplace
	pluginAddInput    textinput.Model
	pluginCloning     bool

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
	settingPluginsPath
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

	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".skill-tui")
	pluginStore := plugin.NewStore(configDir, cfg.PluginsPath)

	search := components.NewSearch(theme)

	app := AppModel{
		cfg:            cfg,
		theme:          theme,
		registry:       registry,
		tabs:           tabNames,
		selected:       make(map[string]bool),
		platforms:      platforms,
		platformMap:    platformMap,
		search:         search,
		pluginStore:    pluginStore,
		pluginSelected: make(map[string]bool),
	}
	app.loadSkills()
	app.loadPlugins()
	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case marketplaceClonedMsg:
		m.pluginCloning = false
		m.currentView = viewList
		if msg.err != nil {
			m.err = msg.err
		}
		m.loadPlugins()
		return m, nil

	case marketplaceInstalledMsg:
		m.pluginCloning = false
		m.currentView = viewList
		m.detailMarketplace = nil
		if msg.err != nil {
			m.err = msg.err
		}
		m.loadPlugins()
		return m, nil

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

		if m.currentView == viewPluginDetail {
			return m.handleMarketplaceDetail(msg)
		}

		if m.currentView == viewPluginInstall {
			return m.handleMarketplaceInstall(msg)
		}

		if m.currentView == viewPluginAdd {
			if m.pluginCloning {
				if msg.String() == "esc" {
					m.pluginCloning = false
					m.currentView = viewList
				}
				return m, nil
			}
			return m.handlePluginAdd(msg)
		}

		if m.activeTab == tabPlugin && m.currentView == viewList {
			return m.handlePluginList(msg)
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
	case tabPlugin:
		return m.renderPluginTab()
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
		{"Plugins Path", m.cfg.PluginsPath},
	}

	for i, item := range items {
		cursor := "  "
		if i == m.settingsCursor {
			cursor = m.theme.Cursor.Render("> ")
		}

		label := m.theme.Accent.Render(fmt.Sprintf("%-14s", item.label))

		if m.settingsEditing && i == m.settingsCursor && (i == settingSkillsPath || i == settingPluginsPath) {
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
				case settingSkillsPath, settingPluginsPath:
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
			switch m.settingsCursor {
			case settingSkillsPath:
				m.cfg.SkillsPath = m.settingsInput.Value()
			case settingPluginsPath:
				m.cfg.PluginsPath = m.settingsInput.Value()
			}
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
		case settingPluginsPath:
			ti := textinput.New()
			ti.SetValue(m.cfg.PluginsPath)
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
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".skill-tui")
	m.pluginStore = plugin.NewStore(configDir, m.cfg.PluginsPath)
	m.loadPlugins()
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

func (m *AppModel) loadPlugins() {
	local, err := m.pluginStore.ScanMarketplaces()
	if err != nil {
		local = nil
	}

	// try fetch remote registry
	ctx, cancel := contextTimeout()
	defer cancel()
	remote, err := plugin.NewRegistryClient().FetchAvailable(ctx)
	if err == nil {
		local = plugin.MergeMarketplaces(local, remote)
	}

	m.marketplaces = local
}

func contextTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (m AppModel) renderPluginTab() string {
	switch m.currentView {
	case viewPluginDetail:
		return m.renderPluginDetail()
	case viewPluginInstall:
		return m.multiSel.View()
	case viewPluginAdd:
		return m.renderPluginAdd()
	default:
		return m.renderPluginList()
	}
}

func (m AppModel) renderPluginList() string {
	var b strings.Builder

	b.WriteString(m.theme.Title.Render("Plugins"))
	b.WriteString(m.theme.Dimmed.Render("  " + m.pluginStore.PluginsDir()))
	b.WriteString("  ")
	b.WriteString(m.search.View())
	b.WriteString("\n\n")

	if len(m.marketplaces) == 0 {
		b.WriteString(m.theme.Dimmed.Render("  No plugins found"))
		b.WriteString("\n")
		b.WriteString(m.theme.Dimmed.Render("  Press 'a' to add marketplace from GitHub  'r' to refresh from registry"))
		return b.String()
	}

	var platCols []string
	for _, p := range m.platforms {
		if p.Category != "central" && p.Installed {
			platCols = append(platCols, p.Name)
		}
	}
	sort.Strings(platCols)

	const nameWidth = 24
	const colWidth = 8

	header := fmt.Sprintf("     %-*s", nameWidth, "")
	for _, p := range platCols {
		header += fmt.Sprintf("%-*s", colWidth, abbreviatePlatform(p))
	}
	b.WriteString(m.theme.Subtitle.Render(header))
	b.WriteString("\n")

	sepLen := 5 + nameWidth + colWidth*len(platCols)
	b.WriteString(m.theme.Dimmed.Render("  " + strings.Repeat("─", sepLen-2)))
	b.WriteString("\n")

	visible := m.visiblePluginRows()
	start := m.pluginScroll
	end := start + visible
	if end > len(m.marketplaces) {
		end = len(m.marketplaces)
	}

	for i := start; i < end; i++ {
		mp := m.marketplaces[i]
		cursor := " "
		if i == m.pluginCursor {
			cursor = m.theme.Cursor.Render(">")
		}

		var check string
		switch mp.Status {
		case "cloned":
			check = m.theme.Success.Render("✓")
		case "missing":
			check = m.theme.Warning.Render("!")
		default:
			check = m.theme.Dimmed.Render("·")
		}

		displayName := mp.Name
		if len(displayName) > nameWidth-1 {
			displayName = displayName[:nameWidth-4] + "..."
		}

		var nameStyled string
		if i == m.pluginCursor {
			nameStyled = m.theme.Selected.Render(fmt.Sprintf("%-*s", nameWidth, displayName))
		} else {
			nameStyled = m.theme.Normal.Render(fmt.Sprintf("%-*s", nameWidth, displayName))
		}

		var cols strings.Builder
		for _, pl := range platCols {
			isInstalled := false
			for _, pp := range m.platforms {
				if pp.Name == pl {
					isInstalled = platform.IsPluginInstalled(pp, mp.Name)
					break
				}
			}
			if isInstalled {
				cols.WriteString(m.theme.Success.Render(fmt.Sprintf("%-*s", colWidth, "✓")))
			} else {
				cols.WriteString(m.theme.Dimmed.Render(fmt.Sprintf("%-*s", colWidth, "·")))
			}
		}

		desc := ""
		if mp.Description != "" {
			d := mp.Description
			if len(d) > 35 {
				d = d[:32] + "..."
			}
			desc = m.theme.Dimmed.Render(d)
		}

		b.WriteString(fmt.Sprintf(" %s %s %s%s %s\n", cursor, check, nameStyled, cols.String(), desc))
	}

	if len(m.marketplaces) > visible {
		b.WriteString(m.theme.Dimmed.Render(fmt.Sprintf("  ─── %d-%d / %d ───", start+1, end, len(m.marketplaces))))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString(m.theme.Dimmed.Render("  ↑/k↓/j: navigate  a: add  i: install  u: uninstall  x: delete  Enter/d: detail  /: search  r: refresh"))

	return b.String()
}

func (m AppModel) visiblePluginRows() int {
	overhead := 8
	rows := m.height - overhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m AppModel) renderPluginDetail() string {
	if m.detailMarketplace == nil {
		return ""
	}
	mp := m.detailMarketplace
	var b strings.Builder

	b.WriteString(m.theme.Title.Render(mp.Name))
	b.WriteString("\n")

	if mp.Version != "" {
		b.WriteString(m.theme.Accent.Render("Version: "))
		b.WriteString(m.theme.Normal.Render(mp.Version))
		b.WriteString("  ")
	}
	if mp.Author != "" {
		b.WriteString(m.theme.Accent.Render("Author: "))
		b.WriteString(m.theme.Normal.Render(mp.Author))
		b.WriteString("  ")
	}
	if mp.RepoURL != "" {
		b.WriteString(m.theme.Accent.Render("Repo: "))
		b.WriteString(m.theme.Normal.Render(mp.RepoURL))
	}
	if mp.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.theme.Normal.Render(mp.Description))
	}
	b.WriteString("\n\n")

	statusLabel := "Available"
	statusStyle := m.theme.Dimmed
	if mp.Status == "cloned" {
		statusLabel = "Cloned"
		statusStyle = m.theme.Success
	}
	b.WriteString(m.theme.Accent.Render("Status: "))
	b.WriteString(statusStyle.Render(statusLabel))
	b.WriteString("\n\n")

	if len(mp.Tags) > 0 {
		b.WriteString(m.theme.Subtitle.Render("Tags"))
		b.WriteString("\n")
		b.WriteString(m.theme.Dimmed.Render("  " + strings.Join(mp.Tags, ", ")))
		b.WriteString("\n\n")
	}

	// Show plugins contained in this marketplace
	if len(mp.Plugins) > 0 {
		b.WriteString(m.theme.Subtitle.Render(fmt.Sprintf("Plugins (%d)", len(mp.Plugins))))
		b.WriteString("\n")
		for _, pi := range mp.Plugins {
			b.WriteString(m.theme.Normal.Render("  • "))
			b.WriteString(m.theme.Normal.Render(pi.Name))
			if pi.Description != "" {
				b.WriteString(m.theme.Dimmed.Render(" - " + pi.Description))
			}
			b.WriteString("\n")
			if len(pi.Commands) > 0 {
				b.WriteString(m.theme.Dimmed.Render("    Commands: " + strings.Join(pi.Commands, ", ")))
				b.WriteString("\n")
			}
			if len(pi.Skills) > 0 {
				b.WriteString(m.theme.Dimmed.Render("    Skills: " + strings.Join(pi.Skills, ", ")))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Show which platforms this marketplace is installed to
	var installedPlatforms []string
	for _, pl := range m.platforms {
		if pl.Category == "central" {
			continue
		}
		if platform.IsPluginInstalled(pl, mp.Name) {
			installedPlatforms = append(installedPlatforms, pl.Name)
		}
	}
	if len(installedPlatforms) > 0 {
		b.WriteString(m.theme.Subtitle.Render("Installed Platforms"))
		b.WriteString("\n")
		for _, ip := range installedPlatforms {
			b.WriteString(m.theme.Success.Render("  ✓ "))
			b.WriteString(m.theme.Normal.Render(ip))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.theme.Dimmed.Render("  Esc: back  i: install to platforms  u: uninstall  x: delete"))

	return b.String()
}

func (m AppModel) handlePluginList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.pluginCursor > 0 {
			m.pluginCursor--
		}
		m.adjustPluginScroll()
	case "down", "j":
		if m.pluginCursor < len(m.marketplaces)-1 {
			m.pluginCursor++
		}
		m.adjustPluginScroll()
	case " ":
		if len(m.marketplaces) > 0 {
			name := m.marketplaces[m.pluginCursor].Name
			m.pluginSelected[name] = !m.pluginSelected[name]
		}
	case "enter", "d":
		if len(m.marketplaces) > 0 {
			mp := m.marketplaces[m.pluginCursor]
			m.detailMarketplace = &mp
			m.currentView = viewPluginDetail
		}
	case "i":
		if len(m.marketplaces) > 0 {
			mp := m.marketplaces[m.pluginCursor]
			m.showMarketplaceInstall(&mp)
		}
	case "x":
		if len(m.marketplaces) > 0 {
			m.deleteMarketplace(m.marketplaces[m.pluginCursor].Name)
		}
	case "u":
		if len(m.marketplaces) > 0 {
			m.uninstallPlatformLinks(m.marketplaces[m.pluginCursor].Name)
		}
	case "r":
		m.loadPlugins()
		m.pluginCursor = 0
		m.pluginScroll = 0
	case "a":
		ti := textinput.New()
		ti.Placeholder = "owner/repo or full URL (e.g. affaan-m/ECC)"
		ti.Focus()
		ti.CharLimit = 200
		ti.Width = 50
		ti.PromptStyle = m.theme.Accent
		ti.TextStyle = m.theme.Normal
		m.pluginAddInput = ti
		m.currentView = viewPluginAdd
	}
	return m, nil
}

func (m *AppModel) adjustPluginScroll() {
	visible := m.visiblePluginRows()
	if m.pluginCursor < m.pluginScroll {
		m.pluginScroll = m.pluginCursor
	}
	if m.pluginCursor >= m.pluginScroll+visible {
		m.pluginScroll = m.pluginCursor - visible + 1
	}
	maxScroll := len(m.marketplaces) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.pluginScroll > maxScroll {
		m.pluginScroll = maxScroll
	}
}

func (m AppModel) handleMarketplaceDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.currentView = viewList
		m.detailMarketplace = nil
	case "i":
		if m.detailMarketplace != nil {
			m.showMarketplaceInstall(m.detailMarketplace)
		}
	case "u":
		if m.detailMarketplace != nil {
			m.uninstallPlatformLinks(m.detailMarketplace.Name)
			for i := range m.marketplaces {
				if m.marketplaces[i].Name == m.detailMarketplace.Name {
					mp := m.marketplaces[i]
					m.detailMarketplace = &mp
					break
				}
			}
		}
	case "x":
		if m.detailMarketplace != nil {
			m.deleteMarketplace(m.detailMarketplace.Name)
			m.detailMarketplace = nil
			m.currentView = viewList
		}
	}
	return m, nil
}

func (m *AppModel) showMarketplaceInstall(mp *plugin.Marketplace) {
	var items []components.MultiSelectItem
	for _, pl := range m.platforms {
		if pl.Category == "central" || !pl.Installed {
			continue
		}
		items = append(items, components.MultiSelectItem{
			Key:   pl.Name,
			Label: pl.Name,
			Desc:  pl.MarketplacesDir,
		})
	}
	if len(items) == 0 {
		return
	}
	m.detailMarketplace = mp
	m.multiSel = components.NewMultiSelect(m.theme, fmt.Sprintf("Install %s to:", mp.Name), items)
	m.currentView = viewPluginInstall
}

func (m AppModel) handleMarketplaceInstall(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.multiSel, cmd = m.multiSel.Update(msg)

	if msg.String() == "enter" {
		selected := m.multiSel.Selected()
		if m.detailMarketplace != nil && len(selected) > 0 {
			m.pluginCloning = true
			mp := *m.detailMarketplace
			store := m.pluginStore

			platList := m.platforms
			return m, func() tea.Msg {
				err := doInstallMarketplaceSync(store, platList, &mp, selected)
				return marketplaceInstalledMsg{marketplace: &mp, err: err}
			}
		}
		m.currentView = viewList
		m.detailMarketplace = nil
		return m, nil
	}

	if msg.String() == "esc" {
		m.currentView = viewPluginDetail
		return m, nil
	}

	return m, cmd
}

func doInstallMarketplaceSync(store *plugin.Store, platList []platform.Platform, mp *plugin.Marketplace, targetPlatforms []string) error {
	// Use the local clone path for marketplace add. This avoids:
	// 1. SSH host key failures from non-interactive exec
	// 2. Clone timeouts for large repos (ECC is ~16MB, takes 2+ min)
	// The repo is already cloned by skill-tui to pluginsDir/<name>.
	localPath := store.PluginDir(mp.Name)

	// Build plugin name list
	var pluginNames []string
	if len(mp.Plugins) > 0 {
		for _, pi := range mp.Plugins {
			pluginNames = append(pluginNames, pi.Name)
		}
	} else {
		pluginNames = []string{mp.Name}
	}

	for _, platName := range targetPlatforms {
		installType := platform.PluginInstallClass(platName)
		if installType == platform.PluginInstallUnsupported {
			continue
		}

		// Use native CLI for platforms that support it (claude, copilot, hermes)
		cli := platform.PlatformCLI(platName)
		if cli != "" {
			if err := platform.InstallMarketplaceViaCLI(platName, localPath, mp.Name, pluginNames); err != nil {
				return fmt.Errorf("installing to %s via CLI: %w", platName, err)
			}
			continue
		}

		// Fallback: platforms without CLI support are skipped
	}

	return nil
}

func (m *AppModel) uninstallPlatformLinks(name string) {
	// Build plugin name list from marketplace
	pluginNames := m.pluginNamesForMarketplace(name)

	for _, pl := range m.platforms {
		if pl.Category == "central" {
			continue
		}
		installType := platform.PluginInstallClass(pl.Name)
		if installType == platform.PluginInstallUnsupported {
			continue
		}

		cli := platform.PlatformCLI(pl.Name)
		if cli != "" {
			platform.UninstallMarketplaceViaCLI(pl.Name, pluginNames)
		}
	}
	m.loadPlugins()
}

func (m *AppModel) deleteMarketplace(name string) {
	m.uninstallPlatformLinks(name)
	m.pluginStore.RemoveMarketplace(name)
	m.loadPlugins()
}

func (m *AppModel) pluginNamesForMarketplace(name string) []string {
	for _, mp := range m.marketplaces {
		if mp.Name == name {
			if len(mp.Plugins) > 0 {
				names := make([]string, len(mp.Plugins))
				for i, pi := range mp.Plugins {
					names[i] = pi.Name
				}
				return names
			}
			return []string{name}
		}
	}
	return []string{name}
}

func (m AppModel) renderPluginAdd() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("Add Marketplace"))
	b.WriteString("\n\n")

	if m.pluginCloning {
		b.WriteString(m.theme.Accent.Render("  Cloning... please wait"))
		b.WriteString("\n\n")
		b.WriteString(m.theme.Dimmed.Render("  Esc: cancel"))
	} else {
		b.WriteString(m.theme.Normal.Render("  Enter a GitHub repo (owner/repo) to add as a marketplace:"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.pluginAddInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.theme.Dimmed.Render("  Install to: " + m.pluginStore.PluginsDir()))
		b.WriteString("\n")
		b.WriteString(m.theme.Dimmed.Render("  Enter: confirm  Esc: cancel"))
	}

	return b.String()
}

func (m AppModel) handlePluginAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = viewList
		m.pluginAddInput.Blur()
		return m, nil
	case "enter":
		repo := strings.TrimSpace(m.pluginAddInput.Value())
		m.pluginAddInput.Blur()
		if repo == "" {
			m.currentView = viewList
			return m, nil
		}
		m.pluginCloning = true
		store := m.pluginStore
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()
			mp, err := store.AddByRepo(ctx, repo)
			return marketplaceClonedMsg{marketplace: mp, err: err}
		}
	default:
		var cmd tea.Cmd
		m.pluginAddInput, cmd = m.pluginAddInput.Update(msg)
		return m, cmd
	}
}

func (m AppModel) renderStatusBar() string {
	m.statusBar = components.StatusBar{
		Theme:     m.theme,
		Width:     m.width,
		Tab:       m.tabs[m.activeTab],
		Platforms: len(m.platforms),
	}

	if m.activeTab == tabPlugin {
		clonedCount := 0
		for _, mp := range m.marketplaces {
			if mp.Status == "cloned" {
				clonedCount++
			}
		}
		m.statusBar.PluginInfo = fmt.Sprintf("%d marketplaces · %d platforms", clonedCount, len(m.platforms))
		m.statusBar.Path = m.pluginStore.PluginsDir()
	} else {
		m.statusBar.SkillCount = len(m.skills)
		m.statusBar.Path = m.cfg.SkillsPath
	}

	return m.statusBar.View()
}
