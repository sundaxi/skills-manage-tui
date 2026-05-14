package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Skill struct {
	Name        string    `json:"name" yaml:"name"`
	Path        string    `json:"path" yaml:"path"`
	Description string    `json:"description" yaml:"description"`
	Version     string    `json:"version" yaml:"version"`
	Author      string    `json:"author" yaml:"author"`
	Tags        []string  `json:"tags" yaml:"tags"`
	Content     string    `json:"content,omitempty" yaml:"content,omitempty"`
	Platforms   []string  `json:"platforms" yaml:"platforms"`
	ModTime     time.Time `json:"mod_time" yaml:"mod_time"`
}

func LoadSkill(skillPath string) (*Skill, error) {
	info, err := os.Stat(skillPath)
	if err != nil {
		return nil, fmt.Errorf("skill path not found: %s", skillPath)
	}

	name := filepath.Base(skillPath)

	var content string
	skillFile := filepath.Join(skillPath, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		skillFile = filepath.Join(skillPath, strings.ToLower(name)+".md")
		data, err = os.ReadFile(skillFile)
		if err != nil {
			skillFile = filepath.Join(skillPath, "skill.md")
			data, err = os.ReadFile(skillFile)
		}
	}

	if err == nil {
		content = string(data)
	}

	meta := parseMetadata(content)

	return &Skill{
		Name:        name,
		Path:        skillPath,
		Description: meta.Description,
		Version:     meta.Version,
		Author:      meta.Author,
		Tags:        meta.Tags,
		Content:     content,
		ModTime:     info.ModTime(),
	}, nil
}

type Registry struct {
	skillsPath string
}

func NewRegistry(skillsPath string) *Registry {
	return &Registry{skillsPath: skillsPath}
}

func (r *Registry) ListSkills() ([]Skill, error) {
	entries, err := os.ReadDir(r.skillsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		skillPath := filepath.Join(r.skillsPath, name)
		skill, err := LoadSkill(skillPath)
		if err != nil {
			continue
		}
		skills = append(skills, *skill)
	}

	return skills, nil
}

func (r *Registry) GetSkill(name string) (*Skill, error) {
	skillPath := filepath.Join(r.skillsPath, name)
	return LoadSkill(skillPath)
}

func (r *Registry) RemoveSkill(name string) error {
	skillPath := filepath.Join(r.skillsPath, name)
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		return fmt.Errorf("skill not found: %s", name)
	}
	return os.RemoveAll(skillPath)
}

func (r *Registry) SkillsPath() string {
	return r.skillsPath
}

func (r *Registry) EnsureDir() error {
	return os.MkdirAll(r.skillsPath, 0755)
}
