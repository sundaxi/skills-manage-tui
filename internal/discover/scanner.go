package discover

import (
	"os"
	"path/filepath"
	"strings"
)

type Discovery struct {
	Name     string
	Path     string
	Platform string
}

var knownSkillDirs = []struct {
	dir      string
	platform string
}{
	{".claude/skills", "Claude Code"},
	{".agents/skills", "Codex CLI"},
	{".cursor/skills", "Cursor"},
	{".gemini/skills", "Gemini CLI"},
	{".windsurf/skills", "Windsurf"},
	{".augment/skills", "Augment"},
	{".copilot/skills", "Copilot"},
	{".aider/skills", "Aider"},
	{".trae/skills", "Trae"},
	{".hermes/skills", "Hermes"},
	{".factory/skills", "Factory Droid"},
	{".kilocode/skills", "KiloCode"},
	{".opencode/skills", "OpenCode"},
	{".amp/skills", "Amp"},
	{".kiro/skills", "Kiro"},
}

func Scan(root string) ([]Discovery, error) {
	var discoveries []Discovery

	for _, known := range knownSkillDirs {
		skillsDir := filepath.Join(root, known.dir)
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}

			discoveries = append(discoveries, Discovery{
				Name:     name,
				Path:     filepath.Join(skillsDir, name),
				Platform: known.platform,
			})
		}
	}

	return discoveries, nil
}

func ScanRecursive(root string, depth int) ([]Discovery, error) {
	var discoveries []Discovery

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) > depth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") && base != "." {
			return nil
		}

		for _, known := range knownSkillDirs {
			expected := filepath.Join(root, known.dir)
			if path == expected {
				entries, err := os.ReadDir(path)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					discoveries = append(discoveries, Discovery{
						Name:     entry.Name(),
						Path:     filepath.Join(path, entry.Name()),
						Platform: known.platform,
					})
				}
			}
		}

		return nil
	})

	return discoveries, err
}
