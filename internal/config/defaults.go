package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type platformsFile struct {
	Platforms []Platform `yaml:"platforms"`
}

func loadDefaultPlatforms(configDir string) ([]Platform, error) {
	platformsPath := findPlatformsFile(configDir)
	if platformsPath == "" {
		return fallbackPlatforms(), nil
	}

	data, err := os.ReadFile(platformsPath)
	if err != nil {
		return fallbackPlatforms(), nil
	}

	var pf platformsFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return fallbackPlatforms(), nil
	}

	home, _ := os.UserHomeDir()
	for i := range pf.Platforms {
		pf.Platforms[i].SkillsDir = expandHome(pf.Platforms[i].SkillsDir, home)
	}

	return pf.Platforms, nil
}

func findPlatformsFile(configDir string) string {
	candidates := []string{
		filepath.Join(configDir, "platforms.yaml"),
	}

	exe, err := os.Executable()
	if err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "configs", "platforms.yaml"))
	}

	candidates = append(candidates,
		"configs/platforms.yaml",
		"/usr/local/share/skill-tui/platforms.yaml",
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func expandHome(path string, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func fallbackPlatforms() []Platform {
	home, _ := os.UserHomeDir()
	expand := func(p string) string { return expandHome(p, home) }

	return []Platform{
		{Name: "claude-code", Category: "coding", SkillsDir: expand("~/.claude/skills/")},
		{Name: "codex-cli", Category: "coding", SkillsDir: expand("~/.agents/skills/")},
		{Name: "cursor", Category: "coding", SkillsDir: expand("~/.cursor/skills/")},
		{Name: "gemini-cli", Category: "coding", SkillsDir: expand("~/.gemini/skills/")},
		{Name: "trae", Category: "coding", SkillsDir: expand("~/.trae/skills/")},
		{Name: "windsurf", Category: "coding", SkillsDir: expand("~/.windsurf/skills/")},
		{Name: "augment", Category: "coding", SkillsDir: expand("~/.augment/skills/")},
		{Name: "copilot", Category: "coding", SkillsDir: expand("~/.copilot/skills/")},
		{Name: "aider", Category: "coding", SkillsDir: expand("~/.aider/skills/")},
		{Name: "hermes", Category: "coding", SkillsDir: expand("~/.hermes/skills/")},
		{Name: "central", Category: "central", SkillsDir: expand("~/.agents/skills/")},
	}
}
