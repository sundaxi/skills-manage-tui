package collection

import (
	"os"
	"path/filepath"
	"testing"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(dir)
}

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/test-config")
	if s.path != "/tmp/test-config/collections.json" {
		t.Errorf("path = %q, want /tmp/test-config/collections.json", s.path)
	}
}

func TestListEmpty(t *testing.T) {
	s := tempStore(t)
	collections, err := s.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if collections != nil {
		t.Errorf("expected nil for empty store, got %v", collections)
	}
}

func TestCreateAndList(t *testing.T) {
	s := tempStore(t)

	err := s.Create("my-collection", "test desc", []string{"skill-a", "skill-b"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	collections, err := s.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}
	if collections[0].Name != "my-collection" {
		t.Errorf("name = %q, want %q", collections[0].Name, "my-collection")
	}
	if collections[0].Description != "test desc" {
		t.Errorf("desc = %q, want %q", collections[0].Description, "test desc")
	}
	if len(collections[0].Skills) != 2 {
		t.Errorf("skills count = %d, want 2", len(collections[0].Skills))
	}
	if collections[0].CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("test", "desc", nil)

	err := s.Create("test", "desc2", nil)
	if err == nil {
		t.Error("expected error for duplicate create")
	}
}

func TestGet(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("alpha", "first", []string{"s1"})
	_ = s.Create("beta", "second", []string{"s2"})

	c, err := s.Get("beta")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if c.Name != "beta" {
		t.Errorf("name = %q, want %q", c.Name, "beta")
	}
	if c.Description != "second" {
		t.Errorf("desc = %q, want %q", c.Description, "second")
	}
}

func TestGetNotFound(t *testing.T) {
	s := tempStore(t)
	_, err := s.Get("missing")
	if err == nil {
		t.Error("expected error for missing collection")
	}
}

func TestDelete(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("to-delete", "will be deleted", nil)
	_ = s.Create("keep", "keep this", nil)

	err := s.Delete("to-delete")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	collections, _ := s.List()
	if len(collections) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(collections))
	}
	if collections[0].Name != "keep" {
		t.Errorf("remaining = %q, want %q", collections[0].Name, "keep")
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := tempStore(t)
	err := s.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for deleting nonexistent collection")
	}
}

func TestAddSkill(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("col", "desc", []string{"s1"})

	err := s.AddSkill("col", "s2")
	if err != nil {
		t.Fatalf("AddSkill error: %v", err)
	}

	c, _ := s.Get("col")
	if len(c.Skills) != 2 {
		t.Fatalf("skills count = %d, want 2", len(c.Skills))
	}
	if c.Skills[1] != "s2" {
		t.Errorf("second skill = %q, want %q", c.Skills[1], "s2")
	}
}

func TestAddSkillDuplicate(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("col", "desc", []string{"s1"})

	err := s.AddSkill("col", "s1")
	if err == nil {
		t.Error("expected error for duplicate skill")
	}
}

func TestAddSkillCollectionNotFound(t *testing.T) {
	s := tempStore(t)
	err := s.AddSkill("missing", "s1")
	if err == nil {
		t.Error("expected error for missing collection")
	}
}

func TestRemoveSkill(t *testing.T) {
	s := tempStore(t)
	_ = s.Create("col", "desc", []string{"s1", "s2", "s3"})

	err := s.RemoveSkill("col", "s2")
	if err != nil {
		t.Fatalf("RemoveSkill error: %v", err)
	}

	c, _ := s.Get("col")
	if len(c.Skills) != 2 {
		t.Fatalf("skills count = %d, want 2", len(c.Skills))
	}
	for _, sk := range c.Skills {
		if sk == "s2" {
			t.Error("s2 should have been removed")
		}
	}
}

func TestRemoveSkillCollectionNotFound(t *testing.T) {
	s := tempStore(t)
	err := s.RemoveSkill("missing", "s1")
	if err == nil {
		t.Error("expected error for missing collection")
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collections.json")
	os.WriteFile(path, []byte("not json"), 0644)

	s := NewStore(dir)
	_, err := s.List()
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "dir")
	s := NewStore(nested)

	err := s.Create("test", "desc", nil)
	if err != nil {
		t.Fatalf("Create in nested dir failed: %v", err)
	}

	_, err = os.Stat(filepath.Join(nested, "collections.json"))
	if err != nil {
		t.Errorf("collections.json not created: %v", err)
	}
}
