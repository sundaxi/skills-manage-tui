package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

func Install(platformSkillsDir, skillPath, skillName string) error {
	if err := os.MkdirAll(platformSkillsDir, 0755); err != nil {
		return fmt.Errorf("creating platform skills dir: %w", err)
	}

	linkPath := filepath.Join(platformSkillsDir, skillName)
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("removing existing link: %w", err)
		}
	}

	absSkillPath, err := filepath.Abs(skillPath)
	if err != nil {
		return fmt.Errorf("resolving skill path: %w", err)
	}

	if err := os.Symlink(absSkillPath, linkPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}

	return nil
}

func Uninstall(platformSkillsDir, skillName string) error {
	linkPath := filepath.Join(platformSkillsDir, skillName)

	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking link: %w", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink, skipping", linkPath)
	}

	return os.Remove(linkPath)
}

func IsLinked(platformSkillsDir, skillName string) bool {
	linkPath := filepath.Join(platformSkillsDir, skillName)
	info, err := os.Lstat(linkPath)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		return false
	}
	_, err = os.Stat(target)
	return err == nil
}

func LinkedPlatforms(cfg map[string]string, skillName string) []string {
	var platforms []string
	for name, dir := range cfg {
		if IsLinked(dir, skillName) {
			platforms = append(platforms, name)
		}
	}
	return platforms
}

func BrokenLinks(platformSkillsDir string) ([]string, error) {
	var broken []string
	entries, err := os.ReadDir(platformSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		linkPath := filepath.Join(platformSkillsDir, entry.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(linkPath)
		if err != nil {
			broken = append(broken, entry.Name())
			continue
		}
		if _, err := os.Stat(target); err != nil {
			broken = append(broken, entry.Name())
		}
	}

	return broken, nil
}
