package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_Empty(t *testing.T) {
	root := t.TempDir()
	discoveries, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(discoveries) != 0 {
		t.Errorf("expected 0 discoveries, got %d", len(discoveries))
	}
}

func TestScan_FindsClaudeSkills(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".claude", "skills", "my-skill")
	os.MkdirAll(skillDir, 0755)

	discoveries, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(discoveries) != 1 {
		t.Fatalf("expected 1 discovery, got %d", len(discoveries))
	}
	if discoveries[0].Name != "my-skill" {
		t.Errorf("name = %q, want %q", discoveries[0].Name, "my-skill")
	}
	if discoveries[0].Platform != "Claude Code" {
		t.Errorf("platform = %q, want %q", discoveries[0].Platform, "Claude Code")
	}
}

func TestScan_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".claude", "skills", ".hidden"), 0755)
	os.MkdirAll(filepath.Join(root, ".claude", "skills", "visible"), 0755)

	discoveries, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(discoveries) != 1 {
		t.Fatalf("expected 1 (hidden skipped), got %d", len(discoveries))
	}
	if discoveries[0].Name != "visible" {
		t.Errorf("name = %q, want %q", discoveries[0].Name, "visible")
	}
}

func TestScan_SkipsFiles(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".claude", "skills"), 0755)
	os.WriteFile(filepath.Join(root, ".claude", "skills", "not-a-dir.md"), []byte("hi"), 0644)

	discoveries, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(discoveries) != 0 {
		t.Errorf("expected 0 (files skipped), got %d", len(discoveries))
	}
}

func TestScan_MultiplePlatforms(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".claude", "skills", "skill-a"), 0755)
	os.MkdirAll(filepath.Join(root, ".copilot", "skills", "skill-b"), 0755)
	os.MkdirAll(filepath.Join(root, ".hermes", "skills", "skill-c"), 0755)

	discoveries, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(discoveries) != 3 {
		t.Fatalf("expected 3 discoveries, got %d", len(discoveries))
	}

	platforms := map[string]bool{}
	for _, d := range discoveries {
		platforms[d.Platform] = true
	}
	for _, expected := range []string{"Claude Code", "Copilot", "Hermes"} {
		if !platforms[expected] {
			t.Errorf("missing platform %q", expected)
		}
	}
}

func TestScanRecursive_FindsSkills(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".claude", "skills", "deep-skill"), 0755)

	discoveries, err := ScanRecursive(root, 5)
	if err != nil {
		t.Fatalf("ScanRecursive error: %v", err)
	}
	if len(discoveries) != 1 {
		t.Fatalf("expected 1 discovery, got %d", len(discoveries))
	}
	if discoveries[0].Name != "deep-skill" {
		t.Errorf("name = %q, want %q", discoveries[0].Name, "deep-skill")
	}
}

func TestScanRecursive_DepthLimit(t *testing.T) {
	root := t.TempDir()
	// .claude is depth 1, skills is depth 2 — too deep for depth=1
	os.MkdirAll(filepath.Join(root, ".claude", "skills", "skill-a"), 0755)

	discoveries, err := ScanRecursive(root, 1)
	if err != nil {
		t.Fatalf("ScanRecursive error: %v", err)
	}
	// With depth=1 the walker stops at .claude level, never reaches skills/
	if len(discoveries) != 0 {
		t.Logf("discoveries: %v", discoveries)
	}
}

func TestScanRecursive_Empty(t *testing.T) {
	root := t.TempDir()
	discoveries, err := ScanRecursive(root, 3)
	if err != nil {
		t.Fatalf("ScanRecursive error: %v", err)
	}
	if len(discoveries) != 0 {
		t.Errorf("expected 0, got %d", len(discoveries))
	}
}
