package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ying-sun1/skill-tui/internal/config"
)

type Platform struct {
	Name            string `json:"name"`
	Category        string `json:"category"`
	SkillsDir       string `json:"skills_dir"`
	CommandsDir     string `json:"commands_dir"`
	MarketplacesDir string `json:"marketplaces_dir"`
	Installed       bool   `json:"installed"`
}

// PluginInstallType classifies how a platform handles plugin installation.
type PluginInstallType string

const (
	// PluginInstallClaude performs full Claude-compatible install:
	// symlink + cache copy + installed_plugins.json + known_marketplaces.json + settings.json.
	PluginInstallClaude PluginInstallType = "claude-native"

	// PluginInstallCopilot uses Claude-compatible flow with additional settings.json fields.
	PluginInstallCopilot PluginInstallType = "copilot-native"

	// PluginInstallUnsupported means this platform has its own incompatible plugin system.
	PluginInstallUnsupported PluginInstallType = "unsupported"

	// PluginInstallSymlinkOnly means this platform only supports skills directory symlink, no plugin system.
	PluginInstallSymlinkOnly PluginInstallType = "symlink-only"
)

// PluginInstallClass returns how a platform handles plugin installation.
func PluginInstallClass(name string) PluginInstallType {
	switch name {
	case "claude-code":
		return PluginInstallClaude
	case "copilot":
		return PluginInstallCopilot
	case "hermes":
		return PluginInstallClaude
	default:
		return PluginInstallSymlinkOnly
	}
}

func ListPlatforms(cfg *config.Config) []Platform {
	var platforms []Platform
	for _, p := range cfg.Platforms {
		installed := dirExists(p.SkillsDir)
		platforms = append(platforms, Platform{
			Name:            p.Name,
			Category:        p.Category,
			SkillsDir:       p.SkillsDir,
			CommandsDir:     p.CommandsDir,
			MarketplacesDir: marketplacesDir(p.SkillsDir),
			Installed:       installed,
		})
	}
	return platforms
}

// marketplacesDir derives the plugin marketplaces directory from SkillsDir.
// e.g. ~/.claude/skills/ → ~/.claude/plugins/marketplaces/
func marketplacesDir(skillsDir string) string {
	parent := filepath.Dir(filepath.Clean(skillsDir))
	return filepath.Join(parent, "plugins", "marketplaces")
}

func DetectInstalled(cfg *config.Config) []Platform {
	var installed []Platform
	for _, p := range ListPlatforms(cfg) {
		if p.Installed {
			installed = append(installed, p)
		}
	}
	return installed
}

func FindPlatform(cfg *config.Config, name string) *Platform {
	for _, p := range ListPlatforms(cfg) {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsPluginInstalled checks if a plugin/marketplace is installed on a platform.
// For Copilot, checks installed-plugins/<marketplace>/<plugin>/ exists.
// For Claude and others, checks if marketplaces/<name>/ directory exists
// (created by the native CLI `plugin marketplace add` command).
func IsPluginInstalled(p Platform, pluginName string) bool {
	installType := PluginInstallClass(p.Name)

	switch installType {
	case PluginInstallCopilot:
		// Copilot: check installed-plugins/<marketplace>/<plugin>/ exists
		installedDir := CopilotInstalledPluginsDir(p.SkillsDir)
		destDir := filepath.Join(installedDir, pluginName, pluginName)
		_, err := os.Stat(destDir)
		return err == nil

	default:
		// Claude + others: check marketplaces/<name>/ directory exists
		dirPath := filepath.Join(p.MarketplacesDir, pluginName)
		_, err := os.Stat(dirPath)
		return err == nil
	}
}

// SymlinkPlugin creates a symlink from marketplacesDir/pluginName → sourceDir.
func SymlinkPlugin(marketplacesDir, pluginName, sourceDir string) error {
	if err := os.MkdirAll(marketplacesDir, 0755); err != nil {
		return fmt.Errorf("creating marketplaces dir: %w", err)
	}
	linkPath := filepath.Join(marketplacesDir, pluginName)
	if info, err := os.Lstat(linkPath); err == nil {
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			// Existing non-symlink directory (e.g. from a previous manual clone)
			if err := os.RemoveAll(linkPath); err != nil {
				return fmt.Errorf("removing existing directory %s: %w", linkPath, err)
			}
		} else {
			os.Remove(linkPath)
		}
	}
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}
	return os.Symlink(absSource, linkPath)
}

// UnsymlinkPlugin removes a plugin symlink from marketplaces directory.
func UnsymlinkPlugin(marketplacesDir, pluginName string) error {
	linkPath := filepath.Join(marketplacesDir, pluginName)
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(linkPath)
	}
	return nil
}

// InstalledMarketplaceNames returns names of marketplaces linked in a platform's dir.
func InstalledMarketplaceNames(marketplacesDir string) []string {
	entries, err := os.ReadDir(marketplacesDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			names = append(names, e.Name())
		}
	}
	return names
}

// --- Claude Code installed_plugins.json management ---

// InstalledPluginEntry represents a single installation record.
type InstalledPluginEntry struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version,omitempty"`
	InstalledAt  string `json:"installedAt"`
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha,omitempty"`
}

// InstalledPluginsFile represents the installed_plugins.json structure.
type InstalledPluginsFile struct {
	Version int                               `json:"version"`
	Plugins map[string][]InstalledPluginEntry `json:"plugins"`
}

// pluginsBaseDir derives the plugins base directory from marketplacesDir.
// e.g. ~/.claude/plugins/marketplaces/ → ~/.claude/plugins/
func pluginsBaseDir(marketplacesDir string) string {
	return filepath.Dir(filepath.Clean(marketplacesDir))
}

// InstalledPluginsPath returns the path to installed_plugins.json for a platform.
func InstalledPluginsPath(marketplacesDir string) string {
	return filepath.Join(pluginsBaseDir(marketplacesDir), "installed_plugins.json")
}

// KnownMarketplacesPath returns the path to known_marketplaces.json for a platform.
func KnownMarketplacesPath(marketplacesDir string) string {
	return filepath.Join(pluginsBaseDir(marketplacesDir), "known_marketplaces.json")
}

// RecordInstalledPlugins adds entries to installed_plugins.json for all plugins
// in the given marketplace. The key format is "marketplaceName@pluginName".
func RecordInstalledPlugins(marketplacesDir, marketplaceName, sourceDir, version, gitSha string, pluginNames []string) error {
	path := InstalledPluginsPath(marketplacesDir)

	file := loadInstalledPlugins(path)
	if file.Plugins == nil {
		file.Plugins = make(map[string][]InstalledPluginEntry)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	absSource, _ := filepath.Abs(sourceDir)

	for _, pluginName := range pluginNames {
		key := marketplaceName + "@" + pluginName
		installPath := filepath.Join(absSource, pluginName)

		entry := InstalledPluginEntry{
			Scope:        "user",
			InstallPath:  installPath,
			Version:      version,
			InstalledAt:  now,
			LastUpdated:  now,
			GitCommitSha: gitSha,
		}
		file.Plugins[key] = []InstalledPluginEntry{entry}
	}

	return saveInstalledPlugins(path, file)
}

// PluginPathInfo holds a plugin's name and its relative path within the marketplace.
type PluginPathInfo struct {
	Name      string // plugin display name
	Path      string // relative path within marketplace source dir
	SourceDir string // override source dir (for external plugins cloned from URL)
}

// RecordInstalledPluginsWithPaths adds entries using the actual plugin subdirectory paths.
func RecordInstalledPluginsWithPaths(marketplacesDir, marketplaceName, sourceDir, version, gitSha string, plugins []PluginPathInfo) error {
	path := InstalledPluginsPath(marketplacesDir)

	file := loadInstalledPlugins(path)
	if file.Plugins == nil {
		file.Plugins = make(map[string][]InstalledPluginEntry)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	absSource, _ := filepath.Abs(sourceDir)

	for _, pi := range plugins {
		key := marketplaceName + "@" + pi.Name
		// Resolve the plugin's actual directory from its relative path
		relPath := strings.TrimPrefix(pi.Path, "./")
		installPath := absSource
		if relPath != "" && relPath != "." {
			installPath = filepath.Join(absSource, relPath)
		}

		entry := InstalledPluginEntry{
			Scope:        "user",
			InstallPath:  installPath,
			Version:      version,
			InstalledAt:  now,
			LastUpdated:  now,
			GitCommitSha: gitSha,
		}
		file.Plugins[key] = []InstalledPluginEntry{entry}
	}

	return saveInstalledPlugins(path, file)
}

// RecordInstalledMarketplace adds a single entry for a whole-marketplace install
// (when the marketplace IS the plugin, e.g. single-plugin repos like ECC).
func RecordInstalledMarketplace(marketplacesDir, marketplaceName, sourceDir, version, gitSha string) error {
	path := InstalledPluginsPath(marketplacesDir)

	file := loadInstalledPlugins(path)
	if file.Plugins == nil {
		file.Plugins = make(map[string][]InstalledPluginEntry)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	absSource, _ := filepath.Abs(sourceDir)

	key := marketplaceName + "@" + marketplaceName
	entry := InstalledPluginEntry{
		Scope:        "user",
		InstallPath:  absSource,
		Version:      version,
		InstalledAt:  now,
		LastUpdated:  now,
		GitCommitSha: gitSha,
	}
	file.Plugins[key] = []InstalledPluginEntry{entry}

	return saveInstalledPlugins(path, file)
}

// RemovePluginCache removes the cached plugin directory for a marketplace.
func RemovePluginCache(marketplacesDir, marketplaceName string) {
	baseDir := pluginsBaseDir(marketplacesDir)
	cacheDir := filepath.Join(baseDir, "cache", marketplaceName)
	os.RemoveAll(cacheDir)
}

// RemoveInstalledPlugin removes entries for a marketplace from installed_plugins.json.
func RemoveInstalledPlugin(marketplacesDir, marketplaceName string) error {
	path := InstalledPluginsPath(marketplacesDir)

	file := loadInstalledPlugins(path)
	if file.Plugins == nil {
		return nil
	}

	prefix := marketplaceName + "@"
	for key := range file.Plugins {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(file.Plugins, key)
		}
	}

	return saveInstalledPlugins(path, file)
}

func loadInstalledPlugins(path string) *InstalledPluginsFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return &InstalledPluginsFile{Version: 2}
	}
	var file InstalledPluginsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return &InstalledPluginsFile{Version: 2}
	}
	return &file
}

func saveInstalledPlugins(path string, file *InstalledPluginsFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// --- Claude Code known_marketplaces.json management ---

// KnownMarketplaceEntry represents an entry in known_marketplaces.json.
type KnownMarketplaceEntry struct {
	Source          KnownMarketplaceSource `json:"source"`
	InstallLocation string                 `json:"installLocation"`
	LastUpdated     string                 `json:"lastUpdated"`
}

// KnownMarketplaceSource represents the source of a marketplace.
type KnownMarketplaceSource struct {
	Source string `json:"source"`
	Repo   string `json:"repo"`
}

// RecordKnownMarketplace adds/updates an entry in known_marketplaces.json.
func RecordKnownMarketplace(marketplacesDir, marketplaceName, repoRef, installLocation string) error {
	path := KnownMarketplacesPath(marketplacesDir)

	entries := loadKnownMarketplaces(path)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	entries[marketplaceName] = KnownMarketplaceEntry{
		Source: KnownMarketplaceSource{
			Source: "github",
			Repo:   repoRef,
		},
		InstallLocation: installLocation,
		LastUpdated:     now,
	}

	return saveKnownMarketplaces(path, entries)
}

// RemoveKnownMarketplace removes an entry from known_marketplaces.json.
func RemoveKnownMarketplace(marketplacesDir, marketplaceName string) error {
	path := KnownMarketplacesPath(marketplacesDir)

	entries := loadKnownMarketplaces(path)
	delete(entries, marketplaceName)

	return saveKnownMarketplaces(path, entries)
}

func loadKnownMarketplaces(path string) map[string]KnownMarketplaceEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]KnownMarketplaceEntry)
	}
	var entries map[string]KnownMarketplaceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return make(map[string]KnownMarketplaceEntry)
	}
	return entries
}

func saveKnownMarketplaces(path string, entries map[string]KnownMarketplaceEntry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// CopyPluginToCache copies a plugin subdirectory into the platform's cache.
// Returns the cache directory path (e.g. ~/.claude/plugins/cache/mpName/pluginName/version/).
func CopyPluginToCache(marketplacesDir, marketplaceName, pluginName, sourcePluginDir, version string) (string, error) {
	baseDir := pluginsBaseDir(marketplacesDir)
	cacheDir := filepath.Join(baseDir, "cache", marketplaceName, pluginName, version)

	if err := os.RemoveAll(cacheDir); err != nil {
		return "", fmt.Errorf("cleaning cache dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
		return "", fmt.Errorf("creating cache parent: %w", err)
	}

	// Copy the plugin directory
	if err := copyDir(sourcePluginDir, cacheDir); err != nil {
		return "", fmt.Errorf("copying plugin to cache: %w", err)
	}

	return cacheDir, nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, si.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Handle symlinks: recreate them as symlinks in dst
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(srcPath)
			if err != nil {
				continue // skip broken symlinks
			}
			os.Symlink(target, dstPath)
			continue
		}

		if entry.IsDir() {
			// Skip .git directories in cache
			if entry.Name() == ".git" {
				continue
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			info, _ := entry.Info()
			mode := os.FileMode(0644)
			if info != nil {
				mode = info.Mode()
			}
			if err := os.WriteFile(dstPath, data, mode); err != nil {
				return err
			}
		}
	}
	return nil
}

// InstallPluginToPlatform performs the full Claude Code compatible plugin install:
// 1. Copy plugin to cache
// 2. Record in installed_plugins.json
// 3. Record in known_marketplaces.json
// Returns the cache install path.
func InstallPluginToPlatform(marketplacesDir, marketplaceName, repoRef string, plugins []PluginPathInfo, sourceDir, version, gitSha string) error {
	absSource, _ := filepath.Abs(sourceDir)

	// Always use git SHA prefix (12 chars) as version, matching Claude Code native behavior.
	// Claude Code uses commit SHA, not semver, for cache paths and version records.
	shortVersion := ""
	if len(gitSha) >= 12 {
		shortVersion = gitSha[:12]
	}
	if shortVersion == "" {
		shortVersion = version // fallback to manifest version if no SHA available
	}

	pluginsFilePath := InstalledPluginsPath(marketplacesDir)
	file := loadInstalledPlugins(pluginsFilePath)
	if file.Plugins == nil {
		file.Plugins = make(map[string][]InstalledPluginEntry)
	}
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	for _, pi := range plugins {
		pluginSourceDir := resolvePluginSourceDir(pi, absSource)

		// Copy to cache
		cachePath, err := CopyPluginToCache(marketplacesDir, marketplaceName, pi.Name, pluginSourceDir, shortVersion)
		if err != nil {
			return fmt.Errorf("caching plugin %s: %w", pi.Name, err)
		}

		// Record in installed_plugins.json
		key := marketplaceName + "@" + pi.Name
		entry := InstalledPluginEntry{
			Scope:        "user",
			InstallPath:  cachePath,
			Version:      shortVersion,
			InstalledAt:  now,
			LastUpdated:  now,
			GitCommitSha: gitSha,
		}
		file.Plugins[key] = []InstalledPluginEntry{entry}
	}

	if err := saveInstalledPlugins(pluginsFilePath, file); err != nil {
		return fmt.Errorf("saving installed_plugins.json: %w", err)
	}

	// Record in known_marketplaces.json
	if repoRef != "" {
		installLoc := filepath.Join(marketplacesDir, marketplaceName)
		if err := RecordKnownMarketplace(marketplacesDir, marketplaceName, repoRef, installLoc); err != nil {
			return fmt.Errorf("saving known_marketplaces.json: %w", err)
		}
	}

	// Enable in settings.json (enabledPlugins + extraKnownMarketplaces)
	pluginsDir := pluginsBaseDir(marketplacesDir)
	platformRoot := filepath.Dir(pluginsDir)
	settingsPath := filepath.Join(platformRoot, "settings.json")
	settings := loadSettingsFile(settingsPath)

	for _, pi := range plugins {
		key := marketplaceName + "@" + pi.Name
		enabled, _ := settings["enabledPlugins"].(map[string]interface{})
		if enabled == nil {
			enabled = make(map[string]interface{})
		}
		enabled[key] = true
		settings["enabledPlugins"] = enabled
	}

	if repoRef != "" {
		known, _ := settings["extraKnownMarketplaces"].(map[string]interface{})
		if known == nil {
			known = make(map[string]interface{})
		}
		parts := strings.SplitN(repoRef, "/", 2)
		if len(parts) == 2 {
			known[marketplaceName] = map[string]interface{}{
				"source": map[string]interface{}{
					"source": "github",
					"repo":   repoRef,
				},
			}
		}
		settings["extraKnownMarketplaces"] = known
	}

	saveSettingsFile(settingsPath, settings)

	return nil
}

// EnablePluginInSettings adds a plugin key to settings.json enabledPlugins.
func EnablePluginInSettings(marketplacesDir, pluginKey string) {
	pluginsDir := pluginsBaseDir(marketplacesDir)
	platformRoot := filepath.Dir(pluginsDir)

	settingsPath := filepath.Join(platformRoot, "settings.json")
	settings := loadSettingsFile(settingsPath)

	enabled, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		enabled = make(map[string]interface{})
	}
	enabled[pluginKey] = true
	settings["enabledPlugins"] = enabled

	saveSettingsFile(settingsPath, settings)
}

// DisablePluginInSettings removes a plugin key from settings.json enabledPlugins.
func DisablePluginInSettings(marketplacesDir, pluginKey string) {
	pluginsDir := pluginsBaseDir(marketplacesDir)
	platformRoot := filepath.Dir(pluginsDir)

	settingsPath := filepath.Join(platformRoot, "settings.json")
	settings := loadSettingsFile(settingsPath)

	enabled, ok := settings["enabledPlugins"].(map[string]interface{})
	if !ok {
		return
	}
	delete(enabled, pluginKey)
	settings["enabledPlugins"] = enabled

	saveSettingsFile(settingsPath, settings)
}

// --- Copilot-specific plugin installation ---

// CopilotInstalledPluginsDir returns the path to Copilot's installed-plugins directory.
// e.g. ~/.copilot/skills/ → ~/.copilot/installed-plugins/
func CopilotInstalledPluginsDir(skillsDir string) string {
	parent := filepath.Dir(filepath.Clean(skillsDir))
	return filepath.Join(parent, "installed-plugins")
}

// CopilotSettingsPath returns the path to Copilot's settings.json.
func CopilotSettingsPath(skillsDir string) string {
	parent := filepath.Dir(filepath.Clean(skillsDir))
	return filepath.Join(parent, "settings.json")
}

// CopilotConfigPath returns the path to Copilot's config.json (auto-managed).
func CopilotConfigPath(skillsDir string) string {
	parent := filepath.Dir(filepath.Clean(skillsDir))
	return filepath.Join(parent, "config.json")
}

// loadCopilotConfig reads Copilot's config.json which has comment lines before the JSON.
// Returns the parsed JSON content and the comment header lines.
func loadCopilotConfig(path string) (map[string]interface{}, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]interface{}), copilotConfigHeader
	}

	content := string(data)
	// Find the first '{' — everything before it is comment header
	idx := strings.Index(content, "{")
	if idx < 0 {
		return make(map[string]interface{}), copilotConfigHeader
	}

	header := content[:idx]
	jsonPart := content[idx:]

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &config); err != nil {
		return make(map[string]interface{}), header
	}
	return config, header
}

const copilotConfigHeader = "// User settings belong in settings.json.\n// This file is managed automatically.\n"

// saveCopilotConfig writes Copilot's config.json preserving the comment header.
func saveCopilotConfig(path string, config map[string]interface{}, header string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(header+string(data)+"\n"), 0644)
}

// InstallPluginToCopilot copies plugin files and updates Copilot's config.json + settings.json.
// Copilot stores:
//   - Plugin files at installed-plugins/<marketplace>/<plugin>/
//   - Install records in config.json → installedPlugins[]
//   - Enable flags in settings.json → enabledPlugins{} + extraKnownMarketplaces{}
func InstallPluginToCopilot(skillsDir, marketplaceName, repoRef string, plugins []PluginPathInfo, sourceDir, version, gitSha string) error {
	absSource, _ := filepath.Abs(sourceDir)

	// Always use git SHA prefix (12 chars) as version, matching native behavior.
	shortVersion := ""
	if len(gitSha) >= 12 {
		shortVersion = gitSha[:12]
	}
	if shortVersion == "" {
		shortVersion = version // fallback to manifest version if no SHA available
	}

	installedDir := CopilotInstalledPluginsDir(skillsDir)
	dataDir := filepath.Join(filepath.Dir(filepath.Clean(skillsDir)), "plugin-data")
	configPath := CopilotConfigPath(skillsDir)
	settingsPath := CopilotSettingsPath(skillsDir)
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// Load config.json for installedPlugins
	config, header := loadCopilotConfig(configPath)

	// Build lookup of existing installedPlugins by marketplace/name
	existingPlugins, _ := config["installedPlugins"].([]interface{})
	pluginMap := make(map[string]map[string]interface{})
	for _, p := range existingPlugins {
		if m, ok := p.(map[string]interface{}); ok {
			mp, _ := m["marketplace"].(string)
			name, _ := m["name"].(string)
			if mp != "" && name != "" {
				pluginMap[mp+"/"+name] = m
			}
		}
	}

	// Load settings.json for enabledPlugins + extraKnownMarketplaces
	settings := loadSettingsFile(settingsPath)

	for _, pi := range plugins {
		pluginSourceDir := resolvePluginSourceDir(pi, absSource)

		// Copy to installed-plugins/<marketplace>/<plugin>/
		destDir := filepath.Join(installedDir, marketplaceName, pi.Name)
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("cleaning copilot plugin dir: %w", err)
		}
		if err := copyDir(pluginSourceDir, destDir); err != nil {
			return fmt.Errorf("copying plugin to copilot: %w", err)
		}

		// Create empty plugin-data dir
		pdDir := filepath.Join(dataDir, marketplaceName, pi.Name)
		os.MkdirAll(pdDir, 0755)

		// Update installedPlugins entry in config.json
		key := marketplaceName + "/" + pi.Name
		pluginMap[key] = map[string]interface{}{
			"name":         pi.Name,
			"marketplace":  marketplaceName,
			"version":      shortVersion,
			"installed_at": now,
			"enabled":      true,
			"cache_path":   destDir,
		}

		// Update enabledPlugins in settings.json
		epKey := marketplaceName + "@" + pi.Name
		enabled, _ := settings["enabledPlugins"].(map[string]interface{})
		if enabled == nil {
			enabled = make(map[string]interface{})
		}
		enabled[epKey] = true
		settings["enabledPlugins"] = enabled
	}

	// Write installedPlugins to config.json
	var newPlugins []interface{}
	for _, p := range pluginMap {
		newPlugins = append(newPlugins, p)
	}
	config["installedPlugins"] = newPlugins

	if err := saveCopilotConfig(configPath, config, header); err != nil {
		return fmt.Errorf("saving copilot config.json: %w", err)
	}

	// Update extraKnownMarketplaces in settings.json
	if repoRef != "" {
		known, _ := settings["extraKnownMarketplaces"].(map[string]interface{})
		if known == nil {
			known = make(map[string]interface{})
		}
		known[marketplaceName] = map[string]interface{}{
			"source": map[string]interface{}{
				"source": "github",
				"repo":   repoRef,
			},
		}
		settings["extraKnownMarketplaces"] = known
	}

	return saveSettingsFile(settingsPath, settings)
}

// UninstallPluginFromCopilot removes plugin files and metadata from Copilot.
func UninstallPluginFromCopilot(skillsDir, marketplaceName string, plugins []PluginPathInfo) error {
	installedDir := CopilotInstalledPluginsDir(skillsDir)
	dataDir := filepath.Join(filepath.Dir(filepath.Clean(skillsDir)), "plugin-data")
	configPath := CopilotConfigPath(skillsDir)
	settingsPath := CopilotSettingsPath(skillsDir)

	os.RemoveAll(filepath.Join(installedDir, marketplaceName))
	os.RemoveAll(filepath.Join(dataDir, marketplaceName))

	// Remove from config.json → installedPlugins[]
	config, header := loadCopilotConfig(configPath)
	existingPlugins, _ := config["installedPlugins"].([]interface{})
	var newPlugins []interface{}
	for _, p := range existingPlugins {
		if m, ok := p.(map[string]interface{}); ok {
			if mp, ok := m["marketplace"].(string); ok && mp == marketplaceName {
				continue
			}
		}
		newPlugins = append(newPlugins, p)
	}
	config["installedPlugins"] = newPlugins
	if err := saveCopilotConfig(configPath, config, header); err != nil {
		return fmt.Errorf("saving copilot config.json: %w", err)
	}

	// Remove from settings.json → enabledPlugins + extraKnownMarketplaces
	settings := loadSettingsFile(settingsPath)

	known, _ := settings["extraKnownMarketplaces"].(map[string]interface{})
	if known != nil {
		delete(known, marketplaceName)
		settings["extraKnownMarketplaces"] = known
	}

	enabled, _ := settings["enabledPlugins"].(map[string]interface{})
	prefix := marketplaceName + "@"
	for k := range enabled {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(enabled, k)
		}
	}
	settings["enabledPlugins"] = enabled

	return saveSettingsFile(settingsPath, settings)
}

func loadSettingsFile(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]interface{})
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return make(map[string]interface{})
	}
	return settings
}

func saveSettingsFile(path string, settings map[string]interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating settings dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// resolvePluginSourceDir returns the actual source directory for a plugin.
// Uses SourceDir override if set, otherwise resolves from Path within baseSource.
func resolvePluginSourceDir(pi PluginPathInfo, baseSource string) string {
	if pi.SourceDir != "" {
		return pi.SourceDir
	}
	relPath := strings.TrimPrefix(pi.Path, "./")
	if relPath == "" || relPath == "." {
		return baseSource
	}
	return filepath.Join(baseSource, relPath)
}

// symlinkPluginSkills creates symlinks from a plugin's skills/ subdirectory
// into the target platform's skills/ directory so the platform can discover them.
func symlinkPluginSkills(pluginDir, targetSkillsDir string) {
	sourceSkillsDir := filepath.Join(pluginDir, "skills")
	entries, err := os.ReadDir(sourceSkillsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		srcPath := filepath.Join(sourceSkillsDir, e.Name())
		linkPath := filepath.Join(targetSkillsDir, e.Name())

		// Skip if already exists (symlink or directory)
		if _, err := os.Stat(linkPath); err == nil {
			continue
		}

		os.Symlink(srcPath, linkPath)
	}
}

// unlinkPluginSkills removes symlinks in targetSkillsDir that point into pluginDir.
func unlinkPluginSkills(pluginDir, targetSkillsDir string) {
	sourceSkillsDir := filepath.Join(pluginDir, "skills")
	entries, err := os.ReadDir(sourceSkillsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		linkPath := filepath.Join(targetSkillsDir, e.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}
		// Only remove if the symlink points into the plugin's skills dir
		if strings.HasPrefix(target, sourceSkillsDir) {
			os.Remove(linkPath)
		}
	}
}
