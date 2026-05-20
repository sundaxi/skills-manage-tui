package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymlinkPluginReplacesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	marketplacesDir := filepath.Join(tmpDir, "marketplaces")
	sourceDir := filepath.Join(tmpDir, "source", "my-plugin")
	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "test.md"), []byte("test"), 0644)

	// Create an existing non-empty directory at the link path (simulates previous manual install)
	existingDir := filepath.Join(marketplacesDir, "my-plugin")
	os.MkdirAll(existingDir, 0755)
	os.WriteFile(filepath.Join(existingDir, "old.md"), []byte("old"), 0644)

	// SymlinkPlugin should replace the existing directory with a symlink
	err := SymlinkPlugin(marketplacesDir, "my-plugin", sourceDir)
	if err != nil {
		t.Fatalf("SymlinkPlugin failed: %v", err)
	}

	// Verify it's now a symlink
	info, err := os.Lstat(filepath.Join(marketplacesDir, "my-plugin"))
	if err != nil {
		t.Fatalf("Lstat failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular entry")
	}

	// Verify symlink points to source
	target, _ := os.Readlink(filepath.Join(marketplacesDir, "my-plugin"))
	absSource, _ := filepath.Abs(sourceDir)
	if target != absSource {
		t.Errorf("symlink target = %s, want %s", target, absSource)
	}
}

func TestCopyDirHandlesSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// Create source with a file and a symlink
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested"), 0644)
	os.Symlink("subdir", filepath.Join(srcDir, "link-to-dir"))
	os.Symlink("file.txt", filepath.Join(srcDir, "link-to-file"))

	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify regular file was copied
	data, err := os.ReadFile(filepath.Join(dstDir, "file.txt"))
	if err != nil || string(data) != "hello" {
		t.Errorf("file.txt not copied correctly")
	}

	// Verify nested file was copied
	data, err = os.ReadFile(filepath.Join(dstDir, "subdir", "nested.txt"))
	if err != nil || string(data) != "nested" {
		t.Errorf("nested.txt not copied correctly")
	}

	// Verify symlinks were recreated
	info, err := os.Lstat(filepath.Join(dstDir, "link-to-dir"))
	if err != nil {
		t.Fatalf("symlink to dir not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for link-to-dir")
	}

	info, err = os.Lstat(filepath.Join(dstDir, "link-to-file"))
	if err != nil {
		t.Fatalf("symlink to file not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink for link-to-file")
	}
}

func TestInstallPluginToPlatform(t *testing.T) {
	// Create a temp directory that mimics ~/.claude/plugins/
	tmpDir := t.TempDir()
	marketplacesDir := filepath.Join(tmpDir, "plugins", "marketplaces")

	// Create a fake plugin source
	sourceDir := filepath.Join(tmpDir, "source", "test-mp")
	pluginSubDir := filepath.Join(sourceDir, "my-plugin")
	os.MkdirAll(filepath.Join(pluginSubDir, ".claude-plugin"), 0755)
	os.MkdirAll(filepath.Join(pluginSubDir, "commands"), 0755)
	os.WriteFile(filepath.Join(pluginSubDir, ".claude-plugin", "plugin.json"), []byte(`{"name":"my-plugin"}`), 0644)
	os.WriteFile(filepath.Join(pluginSubDir, "commands", "test.md"), []byte("# test command"), 0644)

	plugins := []PluginPathInfo{
		{Name: "my-plugin", Path: "./my-plugin"},
	}

	err := InstallPluginToPlatform(marketplacesDir, "test-mp", "owner/repo", plugins, sourceDir, "1.0.0", "abc123def456")
	if err != nil {
		t.Fatalf("InstallPluginToPlatform failed: %v", err)
	}

	// Cache uses version "1.0.0" since it's non-empty
	cacheDir := filepath.Join(tmpDir, "plugins", "cache", "test-mp", "my-plugin", "1.0.0")
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("cache dir not created: %v", err)
	}

	// Verify .claude-plugin/plugin.json in cache
	pluginJSON := filepath.Join(cacheDir, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pluginJSON); err != nil {
		t.Fatalf("plugin.json not in cache: %v", err)
	}

	// Verify installed_plugins.json
	ipData, _ := os.ReadFile(filepath.Join(tmpDir, "plugins", "installed_plugins.json"))
	var ipFile InstalledPluginsFile
	json.Unmarshal(ipData, &ipFile)

	entry, ok := ipFile.Plugins["test-mp@my-plugin"]
	if !ok || len(entry) == 0 {
		t.Fatalf("plugin not in installed_plugins.json")
	}
	if entry[0].InstallPath != cacheDir {
		t.Errorf("installPath = %s, want %s", entry[0].InstallPath, cacheDir)
	}

	// Verify known_marketplaces.json
	kmData, _ := os.ReadFile(filepath.Join(tmpDir, "plugins", "known_marketplaces.json"))
	var km map[string]json.RawMessage
	json.Unmarshal(kmData, &km)
	if _, ok := km["test-mp"]; !ok {
		t.Fatalf("marketplace not in known_marketplaces.json")
	}
}

func TestInstallPluginToCopilot(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillsDir, 0755)

	// Create initial settings.json (user-editable)
	settingsPath := filepath.Join(tmpDir, "settings.json")
	os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{},"extraKnownMarketplaces":{}}`), 0644)

	// Create initial config.json (auto-managed, with comment header)
	configPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(configPath, []byte("// User settings belong in settings.json.\n// This file is managed automatically.\n{\"installedPlugins\":[]}\n"), 0644)

	// Create a fake plugin source
	sourceDir := filepath.Join(tmpDir, "source", "test-mp")
	pluginSubDir := filepath.Join(sourceDir, "my-plugin")
	os.MkdirAll(filepath.Join(pluginSubDir, ".claude-plugin"), 0755)
	os.MkdirAll(filepath.Join(pluginSubDir, "commands"), 0755)
	os.WriteFile(filepath.Join(pluginSubDir, ".claude-plugin", "plugin.json"), []byte(`{"name":"my-plugin"}`), 0644)
	os.WriteFile(filepath.Join(pluginSubDir, "commands", "test.md"), []byte("# test command"), 0644)

	plugins := []PluginPathInfo{
		{Name: "my-plugin", Path: "./my-plugin"},
	}

	err := InstallPluginToCopilot(skillsDir, "test-mp", "owner/repo", plugins, sourceDir, "2.0.0", "abc123def456")
	if err != nil {
		t.Fatalf("InstallPluginToCopilot failed: %v", err)
	}

	// Verify plugin files copied to installed-plugins/
	destDir := filepath.Join(tmpDir, "installed-plugins", "test-mp", "my-plugin")
	if _, err := os.Stat(destDir); err != nil {
		t.Fatalf("installed-plugins dir not created: %v", err)
	}
	pluginJSON := filepath.Join(destDir, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pluginJSON); err != nil {
		t.Fatalf("plugin.json not in installed-plugins: %v", err)
	}

	// Verify plugin-data dir created
	pdDir := filepath.Join(tmpDir, "plugin-data", "test-mp", "my-plugin")
	if _, err := os.Stat(pdDir); err != nil {
		t.Fatalf("plugin-data dir not created: %v", err)
	}

	// Verify config.json has installedPlugins (and preserved comment header)
	configData, _ := os.ReadFile(configPath)
	configStr := string(configData)
	if !strings.Contains(configStr, "// User settings") {
		t.Error("config.json comment header was lost")
	}
	config, _ := loadCopilotConfig(configPath)
	ipList, ok := config["installedPlugins"].([]interface{})
	if !ok || len(ipList) != 1 {
		t.Fatalf("expected 1 installed plugin in config.json, got %v", ipList)
	}
	entry := ipList[0].(map[string]interface{})
	if entry["marketplace"] != "test-mp" || entry["name"] != "my-plugin" {
		t.Errorf("installedPlugins entry = %v", entry)
	}
	if entry["version"] != "2.0.0" {
		t.Errorf("version = %v, want 2.0.0", entry["version"])
	}
	if entry["enabled"] != true {
		t.Errorf("enabled = %v, want true", entry["enabled"])
	}

	// Verify settings.json has enabledPlugins + extraKnownMarketplaces (NOT installedPlugins)
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	// settings.json should NOT have installedPlugins (that goes in config.json)
	if _, hasIP := settings["installedPlugins"]; hasIP {
		t.Error("settings.json should NOT contain installedPlugins — those belong in config.json")
	}

	// Check enabledPlugins
	ep, _ := settings["enabledPlugins"].(map[string]interface{})
	if ep["test-mp@my-plugin"] != true {
		t.Errorf("enabledPlugins missing test-mp@my-plugin: %v", ep)
	}

	// Check extraKnownMarketplaces
	ekm, _ := settings["extraKnownMarketplaces"].(map[string]interface{})
	if _, ok := ekm["test-mp"]; !ok {
		t.Errorf("extraKnownMarketplaces missing test-mp")
	}
}

func TestUninstallPluginFromCopilot(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	os.MkdirAll(skillsDir, 0755)

	cachePath := filepath.Join(tmpDir, "installed-plugins", "test-mp", "my-plugin")

	// Create config.json with existing plugin (auto-managed)
	configPath := filepath.Join(tmpDir, "config.json")
	configContent := "// This file is managed automatically.\n" + `{"installedPlugins":[{"cache_path":"` + cachePath + `","marketplace":"test-mp","name":"my-plugin","version":"2.0.0","enabled":true}]}` + "\n"
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Create settings.json with enabledPlugins + extraKnownMarketplaces
	settingsPath := filepath.Join(tmpDir, "settings.json")
	settingsContent := `{
		"enabledPlugins": {"test-mp@my-plugin": true},
		"extraKnownMarketplaces": {"test-mp": {"source": {"source": "github", "repo": "owner/repo"}}}
	}`
	os.WriteFile(settingsPath, []byte(settingsContent), 0644)

	// Create installed-plugins dir
	os.MkdirAll(cachePath, 0755)

	plugins := []PluginPathInfo{{Name: "my-plugin", Path: "./my-plugin"}}
	err := UninstallPluginFromCopilot(skillsDir, "test-mp", plugins)
	if err != nil {
		t.Fatalf("UninstallPluginFromCopilot failed: %v", err)
	}

	// Verify installed-plugins dir removed
	if _, err := os.Stat(filepath.Join(tmpDir, "installed-plugins", "test-mp")); !os.IsNotExist(err) {
		t.Errorf("installed-plugins/test-mp should be removed")
	}

	// Verify config.json cleaned up
	config, _ := loadCopilotConfig(configPath)
	ipList, _ := config["installedPlugins"].([]interface{})
	if len(ipList) != 0 {
		t.Errorf("config.json installedPlugins should be empty, got %v", ipList)
	}

	// Verify settings.json cleaned up
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	ep, _ := settings["enabledPlugins"].(map[string]interface{})
	if _, ok := ep["test-mp@my-plugin"]; ok {
		t.Errorf("enabledPlugins should not contain test-mp@my-plugin")
	}

	ekm, _ := settings["extraKnownMarketplaces"].(map[string]interface{})
	if _, ok := ekm["test-mp"]; ok {
		t.Errorf("extraKnownMarketplaces should not contain test-mp")
	}
}

func TestPluginInstallClass(t *testing.T) {
	tests := []struct {
		name string
		want PluginInstallType
	}{
		{"claude-code", PluginInstallClaude},
		{"copilot", PluginInstallCopilot},
		{"hermes", PluginInstallClaude},
		{"cursor", PluginInstallSymlinkOnly},
		{"windsurf", PluginInstallSymlinkOnly},
	}
	for _, tt := range tests {
		got := PluginInstallClass(tt.name)
		if got != tt.want {
			t.Errorf("PluginInstallClass(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestEnablePluginInSettingsCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory structure that mimics ~/.copilot/plugins/marketplaces
	marketplacesDir := filepath.Join(tmpDir, "plugins", "marketplaces")
	os.MkdirAll(marketplacesDir, 0755)

	// settings.json doesn't exist yet — EnablePluginInSettings should create it
	EnablePluginInSettings(marketplacesDir, "test-mp:my-plugin")

	settingsPath := filepath.Join(tmpDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json was not created: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON in settings.json: %v", err)
	}

	enabled, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		t.Fatal("enabledPlugins not found in settings.json")
	}

	key := "test-mp:my-plugin"
	val, ok := enabled[key]
	if !ok {
		t.Errorf("plugin key %q not found in enabledPlugins", key)
	}
	if val != true {
		t.Errorf("plugin key %q = %v, want true", key, val)
	}
}
