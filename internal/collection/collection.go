package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Collection struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type Store struct {
	path string
}

func NewStore(configDir string) *Store {
	return &Store{
		path: filepath.Join(configDir, "collections.json"),
	}
}

func (s *Store) load() ([]Collection, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var collections []Collection
	if err := json.Unmarshal(data, &collections); err != nil {
		return nil, err
	}
	return collections, nil
}

func (s *Store) save(collections []Collection) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(collections, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) List() ([]Collection, error) {
	return s.load()
}

func (s *Store) Get(name string) (*Collection, error) {
	collections, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range collections {
		if collections[i].Name == name {
			return &collections[i], nil
		}
	}
	return nil, fmt.Errorf("collection not found: %s", name)
}

func (s *Store) Create(name, description string, skills []string) error {
	collections, err := s.load()
	if err != nil {
		return err
	}

	for _, c := range collections {
		if c.Name == name {
			return fmt.Errorf("collection already exists: %s", name)
		}
	}

	now := time.Now().Format(time.RFC3339)
	collections = append(collections, Collection{
		Name:        name,
		Description: description,
		Skills:      skills,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	return s.save(collections)
}

func (s *Store) Delete(name string) error {
	collections, err := s.load()
	if err != nil {
		return err
	}

	var filtered []Collection
	found := false
	for _, c := range collections {
		if c.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, c)
	}

	if !found {
		return fmt.Errorf("collection not found: %s", name)
	}
	return s.save(filtered)
}

func (s *Store) AddSkill(name, skill string) error {
	collections, err := s.load()
	if err != nil {
		return err
	}

	for i := range collections {
		if collections[i].Name == name {
			for _, s := range collections[i].Skills {
				if s == skill {
					return fmt.Errorf("skill %s already in collection %s", skill, name)
				}
			}
			collections[i].Skills = append(collections[i].Skills, skill)
			collections[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return s.save(collections)
		}
	}
	return fmt.Errorf("collection not found: %s", name)
}

func (s *Store) RemoveSkill(name, skill string) error {
	collections, err := s.load()
	if err != nil {
		return err
	}

	for i := range collections {
		if collections[i].Name == name {
			var filtered []string
			for _, s := range collections[i].Skills {
				if s != skill {
					filtered = append(filtered, s)
				}
			}
			collections[i].Skills = filtered
			collections[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return s.save(collections)
		}
	}
	return fmt.Errorf("collection not found: %s", name)
}
