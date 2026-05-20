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

// AddMarketplaceViaCLI registers a marketplace with the platform CLI.
// For claude/copilot: <cli> plugin marketplace add <source>
// For hermes: creates plugin directory with generated plugin.yaml + __init__.py
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

// InstallPluginViaCLI installs a plugin from a registered marketplace.
// For claude/copilot: <cli> plugin install <plugin>@<marketplace>
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

// InstallMarketplaceViaCLI performs the full marketplace install flow.
func InstallMarketplaceViaCLI(platformName, repoRef string, marketplaceName string, pluginNames []string) error {
	if err := AddMarketplaceViaCLI(platformName, repoRef); err != nil {
		return fmt.Errorf("adding marketplace: %w", err)
	}

	for _, pluginName := range pluginNames {
		if err := InstallPluginViaCLI(platformName, pluginName, marketplaceName); err != nil {
			return fmt.Errorf("installing plugin %s: %w", pluginName, err)
		}
	}

	return nil
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
