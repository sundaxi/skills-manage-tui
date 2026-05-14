package platform

import (
	"os"

	"github.com/ying-sun1/skill-tui/internal/config"
)

type Platform struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	SkillsDir string `json:"skills_dir"`
	Installed bool   `json:"installed"`
}

func ListPlatforms(cfg *config.Config) []Platform {
	var platforms []Platform
	for _, p := range cfg.Platforms {
		installed := dirExists(p.SkillsDir)
		platforms = append(platforms, Platform{
			Name:      p.Name,
			Category:  p.Category,
			SkillsDir: p.SkillsDir,
			Installed: installed,
		})
	}
	return platforms
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
