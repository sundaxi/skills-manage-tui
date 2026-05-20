package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PlatformCLI returns the CLI command name for a platform that supports
// native plugin management, or empty string if not supported.
func PlatformCLI(platformName string) string {
	switch platformName {
	case "claude-code":
		return "claude"
	case "copilot":
		return "copilot"
	case "hermes":
		return "hermes"
	default:
		return ""
	}
}

// ExtractOwnerRepo extracts "owner/repo" from a GitHub URL.
// Returns the original string if it's not a recognized GitHub URL.
func ExtractOwnerRepo(repoURL string) string {
	repoURL = strings.TrimSuffix(strings.TrimSuffix(repoURL, "/"), ".git")
	// Handle https://github.com/owner/repo
	for _, prefix := range []string{
		"https://github.com/",
		"http://github.com/",
		"git@github.com:",
	} {
		if strings.HasPrefix(repoURL, prefix) {
			return repoURL[len(prefix):]
		}
	}
	return repoURL
}

// AddMarketplaceViaCLI registers a marketplace with the platform CLI.
// For claude: claude plugin marketplace add <source>
// For hermes: creates plugin directory with generated plugin.yaml + __init__.py
// Copilot does not need marketplace registration (uses direct install).
func AddMarketplaceViaCLI(platformName, source string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	if cli == "hermes" {
		return hermesCreatePluginDir(source)
	}

	cmd := exec.Command(cli, "plugin", "marketplace", "add", source)
	cmd.Env = append(os.Environ(), "CLAUDE_CODE_PLUGIN_GIT_TIMEOUT_MS=300000")
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "already") {
			return nil
		}
		return fmt.Errorf("%s marketplace add failed: %s: %w", cli, out, err)
	}
	return nil
}

// InstallPluginViaCLI installs a single plugin.
// For claude: claude plugin install <plugin>@<marketplace>
// For copilot: copilot plugin install <source> (owner/repo or plugin@marketplace)
// For hermes: hermes plugins enable <plugin>
func InstallPluginViaCLI(platformName, pluginName, marketplaceName string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	var cmd *exec.Cmd
	if cli == "hermes" {
		cmd = exec.Command(cli, "plugins", "enable", pluginName)
	} else {
		source := pluginName + "@" + marketplaceName
		cmd = exec.Command(cli, "plugin", "install", source)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "already") {
			return nil
		}
		return fmt.Errorf("%s plugin install %s failed: %s: %w", cli, pluginName, out, err)
	}
	return nil
}

// UninstallPluginViaCLI uninstalls a plugin.
// For claude/copilot: <cli> plugin uninstall <plugin>
// For hermes: hermes plugins remove <plugin>
func UninstallPluginViaCLI(platformName, pluginName string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	var cmd *exec.Cmd
	if cli == "hermes" {
		cmd = exec.Command(cli, "plugins", "remove", pluginName)
	} else {
		cmd = exec.Command(cli, "plugin", "uninstall", pluginName)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "not installed") || strings.Contains(out, "not found") || strings.Contains(out, "No such") {
			return nil
		}
		return fmt.Errorf("%s plugin uninstall %s failed: %s: %w", cli, pluginName, out, err)
	}
	return nil
}

// InstallMarketplaceViaCLI performs platform-specific plugin installation.
//
// Claude Code: marketplace add + plugin install (two-step)
// Copilot: direct plugin install from owner/repo (single step, no marketplace needed)
// Hermes: create plugin dir
func InstallMarketplaceViaCLI(platformName, repoSource, localPath, marketplaceName string, pluginNames []string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	switch cli {
	case "copilot":
		return installViaCopilot(repoSource, localPath, pluginNames)
	case "claude":
		return installViaClaude(repoSource, localPath, marketplaceName, pluginNames)
	case "hermes":
		return installViaHermes(localPath, pluginNames)
	}
	return nil
}

// installViaCopilot installs plugins using Copilot's direct install command.
// Copilot supports: copilot plugin install owner/repo
// Falls back to direct file manipulation if CLI is unavailable.
func installViaCopilot(repoSource, localPath string, pluginNames []string) error {
	// Copilot can install directly from owner/repo — no marketplace registration needed
	source := ExtractOwnerRepo(repoSource)

	cmd := exec.Command("copilot", "plugin", "install", source)
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "already") {
			return nil
		}
		return fmt.Errorf("copilot plugin install failed: %s: %w", out, err)
	}
	return nil
}

// installViaClaude uses Claude Code's two-step marketplace flow:
// 1. claude plugin marketplace add <source> (register the marketplace)
// 2. claude plugin install <plugin>@<marketplace> (install each plugin)
// Uses the local path as source since Claude supports "URL, path, or GitHub repo".
func installViaClaude(repoSource, localPath, marketplaceName string, pluginNames []string) error {
	// Use local path if available (avoids re-cloning), fall back to repo URL
	source := localPath
	if source == "" {
		source = repoSource
	}

	if err := AddMarketplaceViaCLI("claude-code", source); err != nil {
		// If local path fails, retry with the repo URL
		if source == localPath && repoSource != "" {
			if err2 := AddMarketplaceViaCLI("claude-code", repoSource); err2 != nil {
				return fmt.Errorf("adding marketplace: %w (also tried repo URL: %v)", err, err2)
			}
		} else {
			return fmt.Errorf("adding marketplace: %w", err)
		}
	}

	for _, pluginName := range pluginNames {
		if err := InstallPluginViaCLI("claude-code", pluginName, marketplaceName); err != nil {
			return fmt.Errorf("installing plugin %s: %w", pluginName, err)
		}
	}
	return nil
}

// installViaHermes creates a hermes plugin directory from the local clone.
func installViaHermes(localPath string, pluginNames []string) error {
	return hermesCreatePluginDir(localPath)
}

// UninstallMarketplaceViaCLI uninstalls all plugins from a marketplace.
func UninstallMarketplaceViaCLI(platformName string, pluginNames []string) error {
	for _, pluginName := range pluginNames {
		if err := UninstallPluginViaCLI(platformName, pluginName); err != nil {
			return fmt.Errorf("uninstalling plugin %s: %w", pluginName, err)
		}
	}
	return nil
}

// hermesCreatePluginDir creates a hermes plugin directory from a local clone.
// Hermes expects plugins at ~/.hermes/plugins/<name>/ with plugin.yaml + __init__.py.
// We symlink the clone content and generate the required manifest files.
func hermesCreatePluginDir(localClonePath string) error {
	name := filepath.Base(localClonePath)
	hermesHome := os.ExpandEnv("$HOME/.hermes")
	pluginDir := filepath.Join(hermesHome, "plugins", name)

	if _, err := os.Stat(pluginDir); err == nil {
		return nil // already exists
	}

	// Create plugin dir and symlink clone content
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("creating hermes plugin dir: %w", err)
	}

	// Read marketplace metadata for description
	desc := readMarketplaceDescription(localClonePath)

	// Generate plugin.yaml
	pluginYaml := fmt.Sprintf("name: %s\nversion: \"1.0.0\"\ndescription: %q\nauthor: marketplace\n", name, desc)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginYaml), 0644); err != nil {
		os.RemoveAll(pluginDir)
		return fmt.Errorf("writing plugin.yaml: %w", err)
	}

	// Generate __init__.py with passthrough register
	initPy := fmt.Sprintf(`"""Hermes adapter for %s marketplace plugin.
Installed by skill-tui from %s.
"""

def register(ctx):
    pass
`, name, localClonePath)
	if err := os.WriteFile(filepath.Join(pluginDir, "__init__.py"), []byte(initPy), 0644); err != nil {
		os.RemoveAll(pluginDir)
		return fmt.Errorf("writing __init__.py: %w", err)
	}

	// Symlink the source clone for access to skills/commands/etc
	linkPath := filepath.Join(pluginDir, "source")
	if err := os.Symlink(localClonePath, linkPath); err != nil {
		os.RemoveAll(pluginDir)
		return fmt.Errorf("symlinking source: %w", err)
	}

	return nil
}

// readMarketplaceDescription reads the description from a marketplace.json file.
func readMarketplaceDescription(clonePath string) string {
	// Try common marketplace manifest locations
	for _, relPath := range []string{
		".claude-plugin/marketplace.json",
		".plugin/marketplace.json",
		".github/plugin/marketplace.json",
		"marketplace.json",
	} {
		data, err := os.ReadFile(filepath.Join(clonePath, relPath))
		if err != nil {
			continue
		}
		// Quick extraction without full JSON parse
		s := string(data)
		if idx := strings.Index(s, `"description"`); idx >= 0 {
			rest := s[idx+len(`"description"`):]
			if colon := strings.IndexByte(rest, ':'); colon >= 0 {
				rest = rest[colon+1:]
				if q1 := strings.IndexByte(rest, '"'); q1 >= 0 {
					rest = rest[q1+1:]
					if q2 := strings.IndexByte(rest, '"'); q2 >= 0 {
						return rest[:q2]
					}
				}
			}
		}
	}
	return "Marketplace plugin managed by skill-tui"
}
