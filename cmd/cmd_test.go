package cmd

import (
	"testing"

	"github.com/ying-sun1/skill-tui/internal/config"
	"github.com/ying-sun1/skill-tui/internal/skill"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "(not set)"},
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"ghp_abcdefghijklmno", "ghp_...lmno"},
	}
	for _, tt := range tests {
		got := maskToken(tt.input)
		if got != tt.want {
			t.Errorf("maskToken(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskIfSecret(t *testing.T) {
	if got := maskIfSecret("github_token", "ghp_test12345678"); got == "ghp_test12345678" {
		t.Error("should mask github_token")
	}
	if got := maskIfSecret("ai_key", "sk-test12345678"); got == "sk-test12345678" {
		t.Error("should mask ai_key")
	}
	if got := maskIfSecret("theme", "mocha"); got != "mocha" {
		t.Errorf("non-secret should not be masked: %q", got)
	}
}

func TestDirExists(t *testing.T) {
	if !dirExists(t.TempDir()) {
		t.Error("temp dir should exist")
	}
	if dirExists("/nonexistent/path") {
		t.Error("nonexistent should not exist")
	}
}

func TestBuildPlatformMap(t *testing.T) {
	cfg := &config.Config{
		Platforms: []config.Platform{
			{Name: "claude-code", Category: "coding", SkillsDir: "/claude"},
			{Name: "central", Category: "central", SkillsDir: "/central"},
			{Name: "copilot", Category: "coding", SkillsDir: "/copilot"},
		},
	}

	m := buildPlatformMap(cfg)
	if len(m) != 2 {
		t.Fatalf("map size = %d, want 2 (central excluded)", len(m))
	}
	if m["claude-code"] != "/claude" {
		t.Errorf("claude-code = %q", m["claude-code"])
	}
	if _, ok := m["central"]; ok {
		t.Error("central should be excluded")
	}
}

func TestFilterByPlatform(t *testing.T) {
	// Create a temp dir with a symlink to simulate a linked skill
	dir := t.TempDir()

	skills := []skill.Skill{
		{Name: "linked"},
		{Name: "not-linked"},
	}

	// No links exist → filter returns empty
	filtered := filterByPlatform(skills, dir)
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered, got %d", len(filtered))
	}
}
