package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ying-sun1/skill-tui/internal/config"
)

func TestInstall(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	skillSrc := filepath.Join(dir, "source", "my-skill")
	os.MkdirAll(skillSrc, 0755)
	os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("hello"), 0644)

	err := Install(skillsDir, skillSrc, "my-skill")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	linkPath := filepath.Join(skillsDir, "my-skill")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("should be a symlink")
	}
}

func TestInstall_ReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	oldSrc := filepath.Join(dir, "old-src")
	newSrc := filepath.Join(dir, "new-src")
	os.MkdirAll(oldSrc, 0755)
	os.MkdirAll(newSrc, 0755)

	os.Symlink(oldSrc, filepath.Join(skillsDir, "skill"))

	err := Install(skillsDir, newSrc, "skill")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	target, _ := os.Readlink(filepath.Join(skillsDir, "skill"))
	absNew, _ := filepath.Abs(newSrc)
	if target != absNew {
		t.Errorf("link target = %q, want %q", target, absNew)
	}
}

func TestUninstall(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0755)

	linkPath := filepath.Join(dir, "my-link")
	os.Symlink(src, linkPath)

	err := Uninstall(dir, "my-link")
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}

	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Error("link should be removed")
	}
}

func TestUninstall_NotASymlink(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real-dir")
	os.MkdirAll(realDir, 0755)

	err := Uninstall(dir, "real-dir")
	if err == nil {
		t.Error("expected error for non-symlink")
	}
}

func TestUninstall_NotExist(t *testing.T) {
	dir := t.TempDir()
	err := Uninstall(dir, "nonexistent")
	if err != nil {
		t.Errorf("should not error for nonexistent: %v", err)
	}
}

func TestIsLinked_True(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0755)
	os.Symlink(src, filepath.Join(dir, "link"))

	if !IsLinked(dir, "link") {
		t.Error("should be linked")
	}
}

func TestIsLinked_False_NotExist(t *testing.T) {
	dir := t.TempDir()
	if IsLinked(dir, "nonexistent") {
		t.Error("should not be linked")
	}
}

func TestIsLinked_False_NotSymlink(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "regular"), 0755)
	if IsLinked(dir, "regular") {
		t.Error("regular dir should not count as linked")
	}
}

func TestIsLinked_False_BrokenLink(t *testing.T) {
	dir := t.TempDir()
	os.Symlink("/nonexistent/target", filepath.Join(dir, "broken"))
	if IsLinked(dir, "broken") {
		t.Error("broken link should not count as linked")
	}
}

func TestLinkedPlatforms(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0755)

	p1Dir := filepath.Join(dir, "p1")
	p2Dir := filepath.Join(dir, "p2")
	p3Dir := filepath.Join(dir, "p3")
	os.MkdirAll(p1Dir, 0755)
	os.MkdirAll(p2Dir, 0755)
	os.MkdirAll(p3Dir, 0755)

	os.Symlink(src, filepath.Join(p1Dir, "skill"))
	os.Symlink(src, filepath.Join(p3Dir, "skill"))

	cfg := map[string]string{
		"platform-1": p1Dir,
		"platform-2": p2Dir,
		"platform-3": p3Dir,
	}

	linked := LinkedPlatforms(cfg, "skill")
	if len(linked) != 2 {
		t.Errorf("linked = %d, want 2", len(linked))
	}
}

func TestBrokenLinks_None(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	os.MkdirAll(src, 0755)
	os.Symlink(src, filepath.Join(dir, "good"))

	broken, err := BrokenLinks(dir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(broken) != 0 {
		t.Errorf("broken = %v, want empty", broken)
	}
}

func TestBrokenLinks_WithBroken(t *testing.T) {
	dir := t.TempDir()
	os.Symlink("/nonexistent/path", filepath.Join(dir, "broken1"))
	os.Symlink("/another/missing", filepath.Join(dir, "broken2"))
	os.MkdirAll(filepath.Join(dir, "not-a-link"), 0755)

	broken, err := BrokenLinks(dir)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(broken) != 2 {
		t.Errorf("broken = %d, want 2: %v", len(broken), broken)
	}
}

func TestBrokenLinks_NonexistentDir(t *testing.T) {
	broken, err := BrokenLinks("/nonexistent/path")
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if broken != nil {
		t.Errorf("should be nil: %v", broken)
	}
}

func TestPlatformCLI(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"claude-code", "claude"},
		{"copilot", "copilot"},
		{"hermes", "hermes"},
		{"cursor", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := PlatformCLI(tt.name)
		if got != tt.want {
			t.Errorf("PlatformCLI(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestIsPluginInstalled_Copilot(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	installedDir := CopilotInstalledPluginsDir(skillsDir)

	// Not installed
	p := Platform{Name: "copilot", SkillsDir: skillsDir}
	if IsPluginInstalled(p, "test-plugin") {
		t.Error("should not be installed")
	}

	// Installed
	os.MkdirAll(filepath.Join(installedDir, "test-plugin", "test-plugin"), 0755)
	if !IsPluginInstalled(p, "test-plugin") {
		t.Error("should be installed")
	}
}

func TestIsPluginInstalled_Claude(t *testing.T) {
	dir := t.TempDir()
	pluginsDir := filepath.Join(dir, "plugins")
	mpDir := filepath.Join(pluginsDir, "marketplaces")
	os.MkdirAll(mpDir, 0755)

	// Not installed (no file)
	p := Platform{Name: "claude-code", MarketplacesDir: mpDir}
	if IsPluginInstalled(p, "my-plugin") {
		t.Error("should not be installed")
	}

	// Write installed_plugins.json
	ipFile := filepath.Join(pluginsDir, "installed_plugins.json")
	os.WriteFile(ipFile, []byte(`{"version":2,"plugins":{"my-plugin@my-plugin":[{"scope":"user"}]}}`), 0644)
	if !IsPluginInstalled(p, "my-plugin") {
		t.Error("should be installed")
	}
}

func TestIsPluginInstalled_Default(t *testing.T) {
	dir := t.TempDir()
	mpDir := filepath.Join(dir, "plugins", "marketplaces")
	os.MkdirAll(mpDir, 0755)

	p := Platform{Name: "cursor", MarketplacesDir: mpDir}

	// Not installed
	if IsPluginInstalled(p, "test") {
		t.Error("should not be installed")
	}

	// Cache exists
	cacheDir := filepath.Join(dir, "plugins", "cache", "test")
	os.MkdirAll(cacheDir, 0755)
	if !IsPluginInstalled(p, "test") {
		t.Error("should be installed (cache)")
	}
}

func TestSymlinkPluginSkills(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	targetDir := filepath.Join(dir, "target")
	os.MkdirAll(filepath.Join(pluginDir, "skills", "skill-a"), 0755)
	os.MkdirAll(filepath.Join(pluginDir, "skills", "skill-b"), 0755)
	os.MkdirAll(filepath.Join(pluginDir, "skills", ".hidden"), 0755)
	os.MkdirAll(targetDir, 0755)

	symlinkPluginSkills(pluginDir, targetDir)

	entries, _ := os.ReadDir(targetDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 symlinks, got %d", len(entries))
	}
}

func TestUnlinkPluginSkills(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	targetDir := filepath.Join(dir, "target")

	skillA := filepath.Join(pluginDir, "skills", "skill-a")
	os.MkdirAll(skillA, 0755)
	os.MkdirAll(targetDir, 0755)

	// Create symlink pointing into plugin
	os.Symlink(skillA, filepath.Join(targetDir, "skill-a"))
	// Create a regular dir (should not be removed)
	os.MkdirAll(filepath.Join(targetDir, "regular"), 0755)
	// Create a symlink pointing elsewhere (should not be removed)
	os.MkdirAll(filepath.Join(dir, "other"), 0755)
	os.Symlink(filepath.Join(dir, "other"), filepath.Join(targetDir, "other-link"))

	unlinkPluginSkills(pluginDir, targetDir)

	entries, _ := os.ReadDir(targetDir)
	// Should have "regular" and "other-link" left
	if len(entries) != 2 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected 2 remaining, got %d: %v", len(entries), names)
	}
}

func TestListPlatforms(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "claude-skills")
	os.MkdirAll(skillsDir, 0755)

	cfg := &config.Config{
		Platforms: []config.Platform{
			{Name: "claude-code", Category: "coding", SkillsDir: skillsDir},
			{Name: "missing", Category: "coding", SkillsDir: filepath.Join(dir, "nope")},
		},
	}

	platforms := ListPlatforms(cfg)
	if len(platforms) != 2 {
		t.Fatalf("platforms = %d, want 2", len(platforms))
	}

	if !platforms[0].Installed {
		t.Error("claude-code should be installed")
	}
	if platforms[1].Installed {
		t.Error("missing should not be installed")
	}
}

func TestDetectInstalled(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	cfg := &config.Config{
		Platforms: []config.Platform{
			{Name: "installed", Category: "coding", SkillsDir: skillsDir},
			{Name: "not-installed", Category: "coding", SkillsDir: "/nonexistent"},
		},
	}

	installed := DetectInstalled(cfg)
	if len(installed) != 1 {
		t.Fatalf("installed = %d, want 1", len(installed))
	}
	if installed[0].Name != "installed" {
		t.Errorf("name = %q", installed[0].Name)
	}
}

func TestFindPlatform(t *testing.T) {
	cfg := &config.Config{
		Platforms: []config.Platform{
			{Name: "claude-code", Category: "coding", SkillsDir: "/test"},
			{Name: "copilot", Category: "coding", SkillsDir: "/test2"},
		},
	}

	p := FindPlatform(cfg, "copilot")
	if p == nil {
		t.Fatal("should find copilot")
	}
	if p.Name != "copilot" {
		t.Errorf("name = %q", p.Name)
	}

	p = FindPlatform(cfg, "nonexistent")
	if p != nil {
		t.Error("should return nil for nonexistent")
	}
}

func TestMarketplacesDir(t *testing.T) {
	tests := []struct {
		skillsDir string
		want      string
	}{
		{"/home/user/.claude/skills/", "/home/user/.claude/plugins/marketplaces"},
		{"/home/user/.claude/skills", "/home/user/.claude/plugins/marketplaces"},
	}
	for _, tt := range tests {
		got := marketplacesDir(tt.skillsDir)
		if got != tt.want {
			t.Errorf("marketplacesDir(%q) = %q, want %q", tt.skillsDir, got, tt.want)
		}
	}
}

func TestReadMarketplaceDescription(t *testing.T) {
	dir := t.TempDir()

	// No marketplace.json
	desc := readMarketplaceDescription(dir)
	if desc != "Marketplace plugin managed by skill-tui" {
		t.Errorf("default desc = %q", desc)
	}

	// With marketplace.json
	mpDir := filepath.Join(dir, ".claude-plugin")
	os.MkdirAll(mpDir, 0755)
	os.WriteFile(filepath.Join(mpDir, "marketplace.json"), []byte(`{
		"name": "test",
		"description": "Test marketplace"
	}`), 0644)

	desc = readMarketplaceDescription(dir)
	if desc != "Test marketplace" {
		t.Errorf("desc = %q, want %q", desc, "Test marketplace")
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	if !dirExists(dir) {
		t.Error("temp dir should exist")
	}
	if dirExists(filepath.Join(dir, "nope")) {
		t.Error("nonexistent should not exist")
	}
}

func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/affaan-m/ECC", "affaan-m/ECC"},
		{"https://github.com/affaan-m/ECC/", "affaan-m/ECC"},
		{"https://github.com/affaan-m/ECC.git", "affaan-m/ECC"},
		{"http://github.com/owner/repo", "owner/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"owner/repo", "owner/repo"},
		{"https://example.com/path", "https://example.com/path"},
		{"/local/path/to/dir", "/local/path/to/dir"},
	}
	for _, tt := range tests {
		got := ExtractOwnerRepo(tt.input)
		if got != tt.want {
			t.Errorf("ExtractOwnerRepo(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCopilotInstalledPluginsDir(t *testing.T) {
	got := CopilotInstalledPluginsDir("/home/user/.copilot/skills")
	want := "/home/user/.copilot/installed-plugins"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCopilotSettingsPath(t *testing.T) {
	got := CopilotSettingsPath("/home/user/.copilot/skills")
	want := "/home/user/.copilot/settings.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCopilotConfigPath(t *testing.T) {
	got := CopilotConfigPath("/home/user/.copilot/skills")
	want := "/home/user/.copilot/config.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadCopilotConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	// Non-existent file returns empty with default header
	cfg, header := loadCopilotConfig(configPath)
	if len(cfg) != 0 {
		t.Error("expected empty config for missing file")
	}
	if header != copilotConfigHeader {
		t.Errorf("expected default header, got %q", header)
	}

	// Valid config with header
	content := "// comment line\n{\"installedPlugins\": []}"
	os.WriteFile(configPath, []byte(content), 0644)
	cfg, header = loadCopilotConfig(configPath)
	if header != "// comment line\n" {
		t.Errorf("header = %q, want %q", header, "// comment line\n")
	}
	if _, ok := cfg["installedPlugins"]; !ok {
		t.Error("should have installedPlugins key")
	}
}

func TestSaveCopilotConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg := map[string]interface{}{
		"installedPlugins": []interface{}{},
	}
	err := saveCopilotConfig(configPath, cfg, "// header\n")
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	data, _ := os.ReadFile(configPath)
	content := string(data)
	if content[:10] != "// header\n" {
		t.Errorf("header not preserved: %q", content[:20])
	}
}

func TestInstallPluginToCopilot_Simple(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	// Create source plugin with a skill file
	sourceDir := filepath.Join(dir, "source")
	os.MkdirAll(filepath.Join(sourceDir, "skills", "my-skill"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "skills", "my-skill", "SKILL.md"), []byte("test"), 0644)

	plugins := []PluginPathInfo{{Name: "test-plugin", Path: "."}}

	err := InstallPluginToCopilot(skillsDir, "test-mp", "owner/repo", plugins, sourceDir, "1.0.0", "abc123def456")
	if err != nil {
		t.Fatalf("install error: %v", err)
	}

	// Verify files were copied
	installedDir := CopilotInstalledPluginsDir(skillsDir)
	destDir := filepath.Join(installedDir, "test-mp", "test-plugin")
	if _, err := os.Stat(destDir); err != nil {
		t.Errorf("plugin dir not created: %v", err)
	}

	// Verify config.json has installedPlugins
	configPath := CopilotConfigPath(skillsDir)
	data, _ := os.ReadFile(configPath)
	if len(data) == 0 {
		t.Error("config.json not written")
	}

	// Verify settings.json has enabledPlugins
	settingsPath := CopilotSettingsPath(skillsDir)
	data, _ = os.ReadFile(settingsPath)
	if len(data) == 0 {
		t.Error("settings.json not written")
	}
}

func TestUninstallPluginFromCopilot_Simple(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillsDir, 0755)

	// Set up installed state
	sourceDir := filepath.Join(dir, "source")
	os.MkdirAll(sourceDir, 0755)
	plugins := []PluginPathInfo{{Name: "test-plugin", Path: "."}}

	InstallPluginToCopilot(skillsDir, "test-mp", "owner/repo", plugins, sourceDir, "1.0.0", "abc123def456")

	// Uninstall
	err := UninstallPluginFromCopilot(skillsDir, "test-mp", plugins)
	if err != nil {
		t.Fatalf("uninstall error: %v", err)
	}

	// Verify plugin dir removed
	installedDir := CopilotInstalledPluginsDir(skillsDir)
	if _, err := os.Stat(filepath.Join(installedDir, "test-mp")); !os.IsNotExist(err) {
		t.Error("plugin dir should be removed")
	}
}
