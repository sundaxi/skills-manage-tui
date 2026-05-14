package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ImportResult struct {
	SkillName string
	Path      string
	Content   string
	Skipped   bool
}

type Importer struct {
	client      *Client
	skillsDir   string
}

func NewImporter(client *Client, skillsDir string) *Importer {
	return &Importer{
		client:    client,
		skillsDir: skillsDir,
	}
}

func (imp *Importer) ImportFromURL(ctx context.Context, repoURL, subPath string) ([]ImportResult, error) {
	repo, err := imp.client.ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	return imp.ImportFromRepo(ctx, repo, subPath)
}

func (imp *Importer) ImportFromRepo(ctx context.Context, repo *RepoInfo, subPath string) ([]ImportResult, error) {
	ref := repo.DefaultBranch
	if ref == "" {
		info, err := imp.client.GetRepo(ctx, repo.Owner, repo.Name)
		if err != nil {
			return nil, fmt.Errorf("fetching repo info: %w", err)
		}
		ref = info.DefaultBranch
	}

	tree, err := imp.client.GetTree(ctx, repo.Owner, repo.Name, ref)
	if err != nil {
		return nil, fmt.Errorf("fetching repo tree: %w", err)
	}

	skillFiles := imp.findSkillFiles(tree, subPath)
	if len(skillFiles) == 0 {
		return nil, fmt.Errorf("no SKILL.md files found in %s (path: %q)", repo.FullName, subPath)
	}

	if err := os.MkdirAll(imp.skillsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating skills dir: %w", err)
	}

	var results []ImportResult
	for _, sf := range skillFiles {
		result, err := imp.importSkillFile(ctx, repo, sf, ref)
		if err != nil {
			results = append(results, ImportResult{
				SkillName: extractSkillName(sf),
				Skipped:   true,
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

func (imp *Importer) findSkillFiles(tree []TreeEntry, subPath string) []TreeEntry {
	var matches []TreeEntry
	for _, entry := range tree {
		if entry.Type != "blob" {
			continue
		}

		name := strings.ToLower(filepath.Base(entry.Path))
		if name != "skill.md" {
			continue
		}

		if subPath != "" && !strings.HasPrefix(entry.Path, subPath) {
			continue
		}

		matches = append(matches, entry)
	}
	return matches
}

func (imp *Importer) importSkillFile(ctx context.Context, repo *RepoInfo, entry TreeEntry, ref string) (*ImportResult, error) {
	skillName := extractSkillName(entry)
	content, err := imp.client.GetFileContent(ctx, repo.Owner, repo.Name, entry.Path, ref)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", entry.Path, err)
	}

	data, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		content = string(data)
	}

	skillDir := filepath.Join(imp.skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return nil, fmt.Errorf("creating skill dir: %w", err)
	}

	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFilePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("writing skill file: %w", err)
	}

	return &ImportResult{
		SkillName: skillName,
		Path:      skillDir,
		Content:   content,
	}, nil
}

func extractSkillName(entry TreeEntry) string {
	dir := filepath.Dir(entry.Path)
	parts := strings.Split(dir, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "." {
		name := parts[len(parts)-1]
		name = strings.TrimPrefix(name, "skill-")
		name = strings.TrimSuffix(name, "-skill")
		return name
	}
	return filepath.Base(dir)
}
