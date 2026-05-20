package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMetadata_Full(t *testing.T) {
	content := `---
description: "A test skill"
version: "1.2.0"
author: "Test Author"
tags: [go, testing, ci]
---
# My Skill

Some content here.
`
	meta := parseMetadata(content)
	if meta.Description != "A test skill" {
		t.Errorf("Description = %q, want %q", meta.Description, "A test skill")
	}
	if meta.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", meta.Version, "1.2.0")
	}
	if meta.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", meta.Author, "Test Author")
	}
	if len(meta.Tags) != 3 {
		t.Fatalf("Tags count = %d, want 3", len(meta.Tags))
	}
	if meta.Tags[0] != "go" || meta.Tags[1] != "testing" || meta.Tags[2] != "ci" {
		t.Errorf("Tags = %v, want [go, testing, ci]", meta.Tags)
	}
}

func TestParseMetadata_Empty(t *testing.T) {
	meta := parseMetadata("")
	if meta.Description != "" || meta.Version != "" || meta.Author != "" {
		t.Errorf("empty content should produce empty metadata, got %+v", meta)
	}
}

func TestParseMetadata_NoFrontmatter(t *testing.T) {
	meta := parseMetadata("Just some text\nwithout frontmatter")
	if meta.Description != "" {
		t.Errorf("no frontmatter should give empty metadata, got desc=%q", meta.Description)
	}
}

func TestParseMetadata_PartialFields(t *testing.T) {
	content := `---
description: "Only desc"
---
Content`
	meta := parseMetadata(content)
	if meta.Description != "Only desc" {
		t.Errorf("Description = %q, want %q", meta.Description, "Only desc")
	}
	if meta.Version != "" {
		t.Errorf("Version should be empty, got %q", meta.Version)
	}
}

func TestExtractFrontmatter_Valid(t *testing.T) {
	content := "---\nkey: value\n---\nbody"
	fm := extractFrontmatter(content)
	if fm != "key: value" {
		t.Errorf("frontmatter = %q, want %q", fm, "key: value")
	}
}

func TestExtractFrontmatter_NoOpeningDelimiter(t *testing.T) {
	fm := extractFrontmatter("no frontmatter here")
	if fm != "" {
		t.Errorf("should be empty, got %q", fm)
	}
}

func TestExtractFrontmatter_NoClosingDelimiter(t *testing.T) {
	fm := extractFrontmatter("---\nkey: value\nno closing")
	// Should return content between --- and EOF
	if fm != "key: value\nno closing" {
		t.Errorf("frontmatter = %q, want %q", fm, "key: value\nno closing")
	}
}

func TestParseTags_Brackets(t *testing.T) {
	tags := parseTags("[go, python, rust]")
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "go" || tags[1] != "python" || tags[2] != "rust" {
		t.Errorf("tags = %v, want [go python rust]", tags)
	}
}

func TestParseTags_Quoted(t *testing.T) {
	tags := parseTags(`["go", "python"]`)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags[0] != "go" || tags[1] != "python" {
		t.Errorf("tags = %v", tags)
	}
}

func TestParseTags_Empty(t *testing.T) {
	tags := parseTags("[]")
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %v", tags)
	}
}

func TestParseTags_SingleItem(t *testing.T) {
	tags := parseTags("golang")
	if len(tags) != 1 || tags[0] != "golang" {
		t.Errorf("tags = %v, want [golang]", tags)
	}
}

func TestLoadSkill_WithSKILLmd(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	os.MkdirAll(skillDir, 0755)

	content := `---
description: "Test skill"
version: "1.0"
author: "tester"
tags: [test]
---
# Test Skill

Does testing things.
`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	s, err := LoadSkill(skillDir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Description != "Test skill" {
		t.Errorf("Description = %q, want %q", s.Description, "Test skill")
	}
	if s.Version != "1.0" {
		t.Errorf("Version = %q, want %q", s.Version, "1.0")
	}
	if s.Author != "tester" {
		t.Errorf("Author = %q, want %q", s.Author, "tester")
	}
	if s.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestLoadSkill_FallbackToSkillMd(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("---\ndescription: fallback\n---\n"), 0644)

	s, err := LoadSkill(skillDir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}
	if s.Description != "fallback" {
		t.Errorf("Description = %q, want %q", s.Description, "fallback")
	}
}

func TestLoadSkill_FallbackToNamedFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "coding")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "coding.md"), []byte("---\ndescription: named\n---\n"), 0644)

	s, err := LoadSkill(skillDir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}
	if s.Description != "named" {
		t.Errorf("Description = %q, want %q", s.Description, "named")
	}
}

func TestLoadSkill_NoSkillFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "empty-skill")
	os.MkdirAll(skillDir, 0755)

	s, err := LoadSkill(skillDir)
	if err != nil {
		t.Fatalf("LoadSkill error: %v", err)
	}
	if s.Name != "empty-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "empty-skill")
	}
	if s.Content != "" {
		t.Errorf("Content should be empty, got %q", s.Content)
	}
}

func TestLoadSkill_PathNotFound(t *testing.T) {
	_, err := LoadSkill("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry("/tmp/skills")
	if r.SkillsPath() != "/tmp/skills" {
		t.Errorf("SkillsPath = %q", r.SkillsPath())
	}
}

func TestRegistry_ListSkills_Empty(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	skills, err := r.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestRegistry_ListSkills_NonexistentDir(t *testing.T) {
	r := NewRegistry("/nonexistent/path")
	skills, err := r.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills should return nil for nonexistent dir, got: %v", err)
	}
	if skills != nil {
		t.Errorf("expected nil, got %v", skills)
	}
}

func TestRegistry_ListSkills_WithSkills(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "skill-a"), 0755)
	os.MkdirAll(filepath.Join(dir, "skill-b"), 0755)
	os.WriteFile(filepath.Join(dir, "not-a-dir.txt"), []byte("x"), 0644)
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)

	r := NewRegistry(dir)
	skills, err := r.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}

func TestRegistry_GetSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	os.MkdirAll(skillDir, 0755)

	r := NewRegistry(dir)
	s, err := r.GetSkill("my-skill")
	if err != nil {
		t.Fatalf("GetSkill error: %v", err)
	}
	if s.Name != "my-skill" {
		t.Errorf("Name = %q", s.Name)
	}
}

func TestRegistry_GetSkill_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)
	_, err := r.GetSkill("missing")
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestRegistry_RemoveSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "to-remove")
	os.MkdirAll(skillDir, 0755)

	r := NewRegistry(dir)
	err := r.RemoveSkill("to-remove")
	if err != nil {
		t.Fatalf("RemoveSkill error: %v", err)
	}

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("skill dir should have been removed")
	}
}

func TestRegistry_RemoveSkill_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)
	err := r.RemoveSkill("missing")
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestRegistry_EnsureDir(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, "new", "skills")
	r := NewRegistry(skillsDir)

	err := r.EnsureDir()
	if err != nil {
		t.Fatalf("EnsureDir error: %v", err)
	}
	if _, err := os.Stat(skillsDir); err != nil {
		t.Errorf("skills dir should exist: %v", err)
	}
}
