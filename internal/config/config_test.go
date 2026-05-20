package config

import (
	"testing"
)

func TestExpandHome(t *testing.T) {
	tests := []struct {
		path string
		home string
		want string
	}{
		{"~/skills", "/home/user", "/home/user/skills"},
		{"~/.agents/skills/", "/Users/test", "/Users/test/.agents/skills"},
		{"/absolute/path", "/home/user", "/absolute/path"},
		{"relative/path", "/home/user", "relative/path"},
		{"", "/home/user", ""},
	}

	for _, tt := range tests {
		got := expandHome(tt.path, tt.home)
		if got != tt.want {
			t.Errorf("expandHome(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
		}
	}
}

func TestSyncCentralPlatform_Exists(t *testing.T) {
	cfg := &Config{
		SkillsPath: "/new/skills/path",
		Platforms: []Platform{
			{Name: "claude", Category: "coding", SkillsDir: "/claude/skills"},
			{Name: "central", Category: "central", SkillsDir: "/old/path"},
		},
	}

	syncCentralPlatform(cfg)

	for _, p := range cfg.Platforms {
		if p.Category == "central" {
			if p.SkillsDir != "/new/skills/path" {
				t.Errorf("central SkillsDir = %q, want %q", p.SkillsDir, "/new/skills/path")
			}
			return
		}
	}
	t.Error("central platform not found")
}

func TestSyncCentralPlatform_Missing(t *testing.T) {
	cfg := &Config{
		SkillsPath: "/my/skills",
		Platforms: []Platform{
			{Name: "claude", Category: "coding", SkillsDir: "/claude"},
		},
	}

	syncCentralPlatform(cfg)

	if len(cfg.Platforms) != 2 {
		t.Fatalf("platforms count = %d, want 2", len(cfg.Platforms))
	}

	last := cfg.Platforms[len(cfg.Platforms)-1]
	if last.Name != "central" || last.Category != "central" || last.SkillsDir != "/my/skills" {
		t.Errorf("added platform = %+v", last)
	}
}

func TestFallbackPlatforms(t *testing.T) {
	platforms := fallbackPlatforms()
	if len(platforms) == 0 {
		t.Fatal("fallbackPlatforms returned empty")
	}

	names := map[string]bool{}
	for _, p := range platforms {
		names[p.Name] = true
		if p.SkillsDir == "" {
			t.Errorf("platform %q has empty SkillsDir", p.Name)
		}
	}

	required := []string{"claude-code", "copilot", "hermes", "central"}
	for _, name := range required {
		if !names[name] {
			t.Errorf("missing required platform %q", name)
		}
	}
}

func TestFindPlatformsFile_NoFile(t *testing.T) {
	path := findPlatformsFile(t.TempDir())
	// Should not panic, returns empty if no file found
	if path != "" {
		// It might find configs/platforms.yaml in the project
		t.Logf("found: %s", path)
	}
}

func TestLoadDefaultPlatforms_NoFile(t *testing.T) {
	platforms, err := loadDefaultPlatforms(t.TempDir())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should return fallback when no file found
	if len(platforms) == 0 {
		t.Error("should return fallback platforms")
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath should not be empty")
	}
}

func TestGetReturnsNilBeforeLoad(t *testing.T) {
	// Reset global
	old := cfg
	cfg = nil
	defer func() { cfg = old }()

	if Get() != nil {
		t.Error("Get() should return nil before Load()")
	}
}

func TestLoad(t *testing.T) {
	// Load uses os.UserHomeDir, so we test the real one
	c, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c == nil {
		t.Fatal("Load returned nil")
	}
	if c.SkillsPath == "" {
		t.Error("SkillsPath should have a default")
	}
	if c.Theme == "" {
		t.Error("Theme should have a default")
	}
	if len(c.Platforms) == 0 {
		t.Error("should have at least fallback platforms")
	}
	// Check central platform is synced
	found := false
	for _, p := range c.Platforms {
		if p.Category == "central" {
			found = true
			if p.SkillsDir != c.SkillsPath {
				t.Errorf("central SkillsDir = %q, want %q", p.SkillsDir, c.SkillsPath)
			}
		}
	}
	if !found {
		t.Error("central platform should be present")
	}
}

func TestReload(t *testing.T) {
	c, err := Reload()
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}
	if c == nil {
		t.Fatal("Reload returned nil")
	}
}

func TestGetAfterLoad(t *testing.T) {
	_, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	c := Get()
	if c == nil {
		t.Error("Get() should return config after Load()")
	}
}
