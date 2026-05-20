package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/config", "/plugins")
	if s.PluginsDir() != "/plugins" {
		t.Errorf("PluginsDir = %q", s.PluginsDir())
	}
	if s.PluginDir("test") != "/plugins/test" {
		t.Errorf("PluginDir = %q", s.PluginDir("test"))
	}
}

func TestNewStore_DefaultPluginsDir(t *testing.T) {
	s := NewStore("/config", "")
	if s.PluginsDir() != "/config/plugins" {
		t.Errorf("PluginsDir = %q, want /config/plugins", s.PluginsDir())
	}
}

func TestLoadCloned_Empty(t *testing.T) {
	s := NewStore(t.TempDir(), t.TempDir())
	records, err := s.loadCloned()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty, got %d", len(records))
	}
}

func TestRecordCloneAndLoad(t *testing.T) {
	configDir := t.TempDir()
	s := NewStore(configDir, t.TempDir())

	mp := Marketplace{Name: "test-mp", RepoURL: "https://github.com/test/repo"}
	if err := s.RecordClone(mp); err != nil {
		t.Fatalf("RecordClone error: %v", err)
	}

	records, _ := s.loadCloned()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records["test-mp"]
	if rec.RepoURL != "https://github.com/test/repo" {
		t.Errorf("RepoURL = %q", rec.RepoURL)
	}
	if rec.Status != "cloned" {
		t.Errorf("Status = %q, want cloned", rec.Status)
	}
	if rec.ClonedAt == "" {
		t.Error("ClonedAt should be set")
	}
}

func TestRemoveRecord(t *testing.T) {
	configDir := t.TempDir()
	s := NewStore(configDir, t.TempDir())

	s.RecordClone(Marketplace{Name: "keep"})
	s.RecordClone(Marketplace{Name: "remove"})

	if err := s.RemoveRecord("remove"); err != nil {
		t.Fatalf("RemoveRecord error: %v", err)
	}

	records, _ := s.loadCloned()
	if len(records) != 1 {
		t.Fatalf("expected 1, got %d", len(records))
	}
	if _, ok := records["keep"]; !ok {
		t.Error("'keep' should remain")
	}
}

func TestRemoveMarketplace(t *testing.T) {
	configDir := t.TempDir()
	pluginsDir := t.TempDir()
	s := NewStore(configDir, pluginsDir)

	// Create marketplace dir and record
	mpDir := filepath.Join(pluginsDir, "test-mp")
	os.MkdirAll(mpDir, 0755)
	os.WriteFile(filepath.Join(mpDir, "README.md"), []byte("hello"), 0644)
	s.RecordClone(Marketplace{Name: "test-mp"})

	if err := s.RemoveMarketplace("test-mp"); err != nil {
		t.Fatalf("RemoveMarketplace error: %v", err)
	}

	// Dir should be gone
	if _, err := os.Stat(mpDir); !os.IsNotExist(err) {
		t.Error("directory should be removed")
	}
	// Record should be gone
	records, _ := s.loadCloned()
	if _, ok := records["test-mp"]; ok {
		t.Error("record should be removed")
	}
}

func TestMergeMarketplaces(t *testing.T) {
	local := []Marketplace{
		{Name: "local-only", Status: "cloned"},
		{Name: "both", Status: "cloned"},
	}
	remote := []Marketplace{
		{Name: "remote-only", Status: "available"},
		{Name: "both", Status: "available"},
	}

	merged := MergeMarketplaces(local, remote)
	if len(merged) != 3 {
		t.Fatalf("merged = %d, want 3", len(merged))
	}

	names := map[string]string{}
	for _, m := range merged {
		names[m.Name] = m.Status
	}
	if names["local-only"] != "cloned" {
		t.Errorf("local-only status = %q", names["local-only"])
	}
	if names["both"] != "cloned" {
		t.Errorf("both should keep local (cloned), got %q", names["both"])
	}
	if names["remote-only"] != "available" {
		t.Errorf("remote-only status = %q", names["remote-only"])
	}
}

func TestMergeMarketplaces_Empty(t *testing.T) {
	merged := MergeMarketplaces(nil, nil)
	if len(merged) != 0 {
		t.Errorf("expected 0, got %d", len(merged))
	}
}

func TestParseCustomManifest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{
		"name": "custom-plugin",
		"description": "A custom plugin",
		"version": "2.0",
		"author": "tester",
		"tags": ["go", "test"]
	}`), 0644)

	mp := parseCustomManifest(dir)
	if mp == nil {
		t.Fatal("expected marketplace, got nil")
	}
	if mp.Name != "custom-plugin" {
		t.Errorf("Name = %q", mp.Name)
	}
	if mp.Version != "2.0" {
		t.Errorf("Version = %q", mp.Version)
	}
	if mp.Author != "tester" {
		t.Errorf("Author = %q", mp.Author)
	}
	if len(mp.Tags) != 2 {
		t.Errorf("Tags = %v", mp.Tags)
	}
}

func TestParseCustomManifest_Missing(t *testing.T) {
	mp := parseCustomManifest(t.TempDir())
	if mp != nil {
		t.Error("expected nil for missing file")
	}
}

func TestParseCustomManifest_Invalid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "plugin.json"), []byte("not json"), 0644)

	mp := parseCustomManifest(dir)
	if mp != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestAutoScanMarketplace(t *testing.T) {
	dir := t.TempDir()
	// Create commands/ and skills/ dirs
	os.MkdirAll(filepath.Join(dir, "commands"), 0755)
	os.MkdirAll(filepath.Join(dir, "skills", "my-skill"), 0755)
	os.WriteFile(filepath.Join(dir, "commands", "run.md"), []byte("# Run"), 0644)

	mp := autoScanMarketplace(dir, "test-auto")
	if mp.Name != "test-auto" {
		t.Errorf("Name = %q", mp.Name)
	}
	if len(mp.Plugins) != 1 {
		t.Fatalf("Plugins = %d", len(mp.Plugins))
	}
	if len(mp.Plugins[0].Commands) != 1 {
		t.Errorf("Commands = %v", mp.Plugins[0].Commands)
	}
	if len(mp.Plugins[0].Skills) != 1 {
		t.Errorf("Skills = %v", mp.Plugins[0].Skills)
	}
}

func TestScanPluginItems(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "commands"), 0755)
	os.MkdirAll(filepath.Join(dir, "skills", "a"), 0755)
	os.MkdirAll(filepath.Join(dir, "skills", "b"), 0755)
	os.MkdirAll(filepath.Join(dir, "skills", ".hidden"), 0755)
	os.WriteFile(filepath.Join(dir, "commands", "cmd1.md"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "commands", "cmd2.md"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "commands", "not-md.txt"), []byte("x"), 0644)

	item := &PluginItem{Name: "test"}
	scanPluginItems(dir, item)

	if len(item.Commands) != 2 {
		t.Errorf("Commands = %v, want 2", item.Commands)
	}
	if len(item.Skills) != 2 {
		t.Errorf("Skills = %v, want 2", item.Skills)
	}
}

func TestScanForPlugins(t *testing.T) {
	dir := t.TempDir()
	// Create plugins/ subdirectory structure
	p1 := filepath.Join(dir, "plugins", "plugin-a")
	os.MkdirAll(filepath.Join(p1, ".claude-plugin"), 0755)
	os.WriteFile(filepath.Join(p1, ".claude-plugin", "plugin.json"), []byte(`{"description":"Plugin A"}`), 0644)
	os.MkdirAll(filepath.Join(p1, "commands"), 0755)
	os.WriteFile(filepath.Join(p1, "commands", "run.md"), []byte("x"), 0644)

	p2 := filepath.Join(dir, "plugins", "plugin-b")
	os.MkdirAll(p2, 0755)

	os.MkdirAll(filepath.Join(dir, "plugins", ".hidden"), 0755)

	mp := &Marketplace{Name: "test"}
	scanForPlugins(dir, mp)

	if len(mp.Plugins) != 2 {
		t.Fatalf("Plugins = %d, want 2", len(mp.Plugins))
	}

	for _, p := range mp.Plugins {
		if p.Name == "plugin-a" {
			if p.Description != "Plugin A" {
				t.Errorf("plugin-a description = %q", p.Description)
			}
			if len(p.Commands) != 1 {
				t.Errorf("plugin-a commands = %v", p.Commands)
			}
		}
	}
}

func TestResolvePluginDir_LocalPath(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(t.TempDir(), dir)

	pi := PluginItem{Name: "test", Path: "./subdir"}
	resolved, err := s.ResolvePluginDir("/source", pi)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resolved != "/source/subdir" {
		t.Errorf("resolved = %q, want /source/subdir", resolved)
	}
}

func TestResolvePluginDir_RootPath(t *testing.T) {
	s := NewStore(t.TempDir(), t.TempDir())

	pi := PluginItem{Name: "test", Path: "."}
	resolved, err := s.ResolvePluginDir("/source", pi)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resolved != "/source" {
		t.Errorf("resolved = %q, want /source", resolved)
	}
}

func TestResolvePluginDir_EmptyPath(t *testing.T) {
	s := NewStore(t.TempDir(), t.TempDir())

	pi := PluginItem{Name: "test", Path: ""}
	resolved, err := s.ResolvePluginDir("/source", pi)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resolved != "/source" {
		t.Errorf("resolved = %q, want /source", resolved)
	}
}
