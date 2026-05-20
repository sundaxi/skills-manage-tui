package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSuperpowersMarketplace(t *testing.T) {
	// Create a temp marketplace directory with superpowers-style marketplace.json
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude-plugin"), 0755)

	marketplaceJSON := `{
  "name": "superpowers-marketplace",
  "owner": {"name": "Test"},
  "metadata": {"description": "Test marketplace", "version": "1.0.0"},
  "plugins": [
    {
      "name": "superpowers",
      "source": {"source": "url", "url": "https://github.com/obra/superpowers.git"},
      "description": "Core skills library"
    },
    {
      "name": "local-plugin",
      "source": "./plugins/local",
      "description": "Local plugin"
    }
  ]
}`
	os.WriteFile(filepath.Join(dir, ".claude-plugin", "marketplace.json"), []byte(marketplaceJSON), 0644)

	mp := parseMarketplaceDir(dir)
	if mp.Name != "superpowers-marketplace" {
		t.Fatalf("expected name 'superpowers-marketplace', got %q", mp.Name)
	}
	if mp.Version != "1.0.0" {
		t.Fatalf("expected version '1.0.0', got %q", mp.Version)
	}
	if len(mp.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(mp.Plugins))
	}

	// Check external URL plugin
	p0 := mp.Plugins[0]
	if p0.Name != "superpowers" {
		t.Fatalf("expected plugin name 'superpowers', got %q", p0.Name)
	}
	if p0.SourceURL != "https://github.com/obra/superpowers.git" {
		t.Fatalf("expected SourceURL 'https://github.com/obra/superpowers.git', got %q", p0.SourceURL)
	}
	if p0.Path != "" {
		t.Fatalf("expected empty Path for URL plugin, got %q", p0.Path)
	}

	// Check local path plugin
	p1 := mp.Plugins[1]
	if p1.Name != "local-plugin" {
		t.Fatalf("expected plugin name 'local-plugin', got %q", p1.Name)
	}
	if p1.SourceURL != "" {
		t.Fatalf("expected empty SourceURL for local plugin, got %q", p1.SourceURL)
	}
	if p1.Path != "./plugins/local" {
		t.Fatalf("expected Path './plugins/local', got %q", p1.Path)
	}
}

func TestParseECCMarketplace(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude-plugin"), 0755)

	marketplaceJSON := `{
  "name": "ecc",
  "owner": {"name": "Test"},
  "metadata": {"description": "ECC plugin"},
  "plugins": [
    {
      "name": "ecc",
      "source": "./",
      "description": "The full ECC plugin"
    }
  ]
}`
	os.WriteFile(filepath.Join(dir, ".claude-plugin", "marketplace.json"), []byte(marketplaceJSON), 0644)

	mp := parseMarketplaceDir(dir)
	if mp.Name != "ecc" {
		t.Fatalf("expected name 'ecc', got %q", mp.Name)
	}
	if len(mp.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(mp.Plugins))
	}
	p0 := mp.Plugins[0]
	if p0.Name != "ecc" {
		t.Fatalf("expected plugin name 'ecc', got %q", p0.Name)
	}
	if p0.Path != "./" {
		t.Fatalf("expected Path './', got %q", p0.Path)
	}
	if p0.SourceURL != "" {
		t.Fatalf("expected empty SourceURL, got %q", p0.SourceURL)
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/obra/superpowers.git", "superpowers"},
		{"https://github.com/obra/superpowers", "superpowers"},
		{"https://github.com/obra/superpowers-chrome", "superpowers-chrome"},
	}
	for _, tt := range tests {
		got := repoNameFromURL(tt.url)
		if got != tt.want {
			t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestScanMarketplacesIncludesMissing(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)

	store := NewStore(configDir, pluginsDir)

	// Record a marketplace in marketplaces.json but don't create its directory.
	mp := Marketplace{
		Name:    "ecc",
		RepoURL: "https://github.com/affaan-m/ECC",
	}
	if err := store.RecordClone(mp); err != nil {
		t.Fatal(err)
	}

	// Create a different marketplace directory on disk.
	goodDir := filepath.Join(pluginsDir, "good-plugin")
	os.MkdirAll(filepath.Join(goodDir, ".claude-plugin"), 0755)
	os.WriteFile(filepath.Join(goodDir, ".claude-plugin", "plugin.json"),
		[]byte(`{"name":"good-plugin","version":"1.0"}`), 0644)

	results, err := store.ScanMarketplaces()
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 marketplaces, got %d", len(results))
	}

	foundGood, foundEcc := false, false
	for _, r := range results {
		switch r.Name {
		case "good-plugin":
			foundGood = true
			if r.Status != "cloned" {
				t.Errorf("good-plugin status = %q, want 'cloned'", r.Status)
			}
		case "ecc":
			foundEcc = true
			if r.Status != "missing" {
				t.Errorf("ecc status = %q, want 'missing'", r.Status)
			}
			if r.RepoURL != "https://github.com/affaan-m/ECC" {
				t.Errorf("ecc RepoURL = %q, want correct URL", r.RepoURL)
			}
		}
	}
	if !foundGood {
		t.Error("good-plugin not found in results")
	}
	if !foundEcc {
		t.Error("ecc (missing) not found in results")
	}
}

func TestAddByRepoCaseInsensitiveRename(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	os.MkdirAll(pluginsDir, 0755)
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)

	// Simulate what AddByRepo does after cloning: directory is named "ECC" (from URL)
	// but manifest says "ecc". Test the rename logic directly.
	eccDir := filepath.Join(pluginsDir, "ECC")
	os.MkdirAll(filepath.Join(eccDir, ".claude-plugin"), 0755)
	os.WriteFile(filepath.Join(eccDir, ".claude-plugin", "plugin.json"),
		[]byte(`{"name":"ecc","version":"1.0"}`), 0644)
	os.WriteFile(filepath.Join(eccDir, ".claude-plugin", "marketplace.json"),
		[]byte(`{"name":"ecc","plugins":[{"name":"ecc","source":"./"}]}`), 0644)
	os.WriteFile(filepath.Join(eccDir, "README.md"), []byte("# ECC"), 0644)

	// Verify directory exists before rename logic
	if _, err := os.Stat(eccDir); err != nil {
		t.Fatalf("eccDir should exist: %v", err)
	}

	// Apply the same rename logic as AddByRepo
	mp := parseMarketplaceDir(eccDir)
	repoName := "ECC"

	if mp.Name != "" && mp.Name != repoName {
		canonicalDir := filepath.Join(pluginsDir, mp.Name)
		if canonicalDir != eccDir {
			if strings.EqualFold(mp.Name, repoName) {
				tmpRenameDir := eccDir + ".tmp-rename"
				if err := os.Rename(eccDir, tmpRenameDir); err == nil {
					if err := os.Rename(tmpRenameDir, canonicalDir); err == nil {
						eccDir = canonicalDir
					} else {
						os.Rename(tmpRenameDir, eccDir)
					}
				}
			} else {
				os.RemoveAll(canonicalDir)
				if err := os.Rename(eccDir, canonicalDir); err == nil {
					eccDir = canonicalDir
				}
			}
		}
	}

	// Verify directory still exists (not deleted by rename)
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, e := range entries {
		if strings.EqualFold(e.Name(), "ecc") {
			found = true
			// Verify contents survived
			readme, err := os.ReadFile(filepath.Join(pluginsDir, e.Name(), "README.md"))
			if err != nil {
				t.Fatalf("README.md should survive rename: %v", err)
			}
			if string(readme) != "# ECC" {
				t.Errorf("README.md content = %q, want '# ECC'", string(readme))
			}
		}
	}
	if !found {
		t.Fatal("ecc directory not found after rename — was destroyed by case-insensitive RemoveAll bug")
	}
}

func TestParseMarketplacePluginSourceTypes(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantPath string
		wantURL  string
	}{
		{
			"string source",
			`{"name":"test","source":"./subdir"}`,
			"./subdir", "",
		},
		{
			"object source with url",
			`{"name":"test","source":{"source":"url","url":"https://github.com/foo/bar.git"}}`,
			"", "https://github.com/foo/bar.git",
		},
		{
			"empty string source",
			`{"name":"test","source":""}`,
			"", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := parseMarketplacePlugin("/tmp", json.RawMessage(tt.json))
			if item.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", item.Path, tt.wantPath)
			}
			if item.SourceURL != tt.wantURL {
				t.Errorf("SourceURL = %q, want %q", item.SourceURL, tt.wantURL)
			}
		})
	}
}
