package platform

import (
	"fmt"
	"os/exec"
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
		return "claude"
	default:
		return ""
	}
}

// AddMarketplaceViaCLI runs: <cli> plugin marketplace add <source>
func AddMarketplaceViaCLI(platformName, source string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	cmd := exec.Command(cli, "plugin", "marketplace", "add", source)
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		// "already exists" is not an error — marketplace was previously registered
		if strings.Contains(out, "already") {
			return nil
		}
		return fmt.Errorf("%s plugin marketplace add failed: %s: %w", cli, out, err)
	}
	return nil
}

// InstallPluginViaCLI runs: <cli> plugin install <plugin>@<marketplace>
// For single-plugin repos where plugin == marketplace, pass the same name for both.
func InstallPluginViaCLI(platformName, pluginName, marketplaceName string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	source := pluginName + "@" + marketplaceName
	cmd := exec.Command(cli, "plugin", "install", source)
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		// "already installed" is not an error
		if strings.Contains(out, "already installed") {
			return nil
		}
		return fmt.Errorf("%s plugin install %s failed: %s: %w", cli, source, out, err)
	}
	return nil
}

// UninstallPluginViaCLI runs: <cli> plugin uninstall <plugin>
func UninstallPluginViaCLI(platformName, pluginName string) error {
	cli := PlatformCLI(platformName)
	if cli == "" {
		return fmt.Errorf("platform %q has no CLI plugin support", platformName)
	}

	cmd := exec.Command(cli, "plugin", "uninstall", pluginName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		out := strings.TrimSpace(string(output))
		// "not installed" is not an error
		if strings.Contains(out, "not installed") || strings.Contains(out, "not found") {
			return nil
		}
		return fmt.Errorf("%s plugin uninstall %s failed: %s: %w", cli, pluginName, out, err)
	}
	return nil
}

// InstallMarketplaceViaCLI performs the full marketplace install flow:
// 1. Register the marketplace: <cli> plugin marketplace add <source>
// 2. Install each plugin: <cli> plugin install <plugin>@<marketplace>
func InstallMarketplaceViaCLI(platformName, repoRef string, marketplaceName string, pluginNames []string) error {
	// Step 1: Register the marketplace
	if err := AddMarketplaceViaCLI(platformName, repoRef); err != nil {
		return fmt.Errorf("adding marketplace: %w", err)
	}

	// Step 2: Install each plugin from the marketplace
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
