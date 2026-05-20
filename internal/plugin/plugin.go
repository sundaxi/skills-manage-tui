package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultRegistryURL = "https://raw.githubusercontent.com/ying-sun1/skill-tui/main/configs/plugins.json"

// Marketplace represents a GitHub repo that acts as a plugin marketplace.
// Each marketplace can contain one or more PluginItems.
type Marketplace struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	RepoURL     string       `json:"repo_url"`
	Version     string       `json:"version"`
	Author      string       `json:"author"`
	Tags        []string     `json:"tags"`
	Plugins     []PluginItem `json:"plugins"`
	Status      string       `json:"status"` // "cloned", "available"
	ClonedAt    string       `json:"cloned_at,omitempty"`
}

// PluginItem represents a single plugin within a marketplace.
type PluginItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`       // local subdirectory path (e.g. "./skills/")
	SourceURL   string   `json:"source_url"` // external git URL when source is a URL reference
	Commands    []string `json:"commands"`
	Skills      []string `json:"skills"`
}

// Store manages marketplace persistence and scanning.
type Store struct {
	pluginsDir    string
	installedPath string
}

func NewStore(configDir string, pluginsPath string) *Store {
	dir := pluginsPath
	if dir == "" {
		dir = filepath.Join(configDir, "plugins")
	}
	return &Store{
		pluginsDir:    dir,
		installedPath: filepath.Join(configDir, "marketplaces.json"),
	}
}

func (s *Store) PluginsDir() string {
	return s.pluginsDir
}

func (s *Store) PluginDir(name string) string {
	return filepath.Join(s.pluginsDir, name)
}

// loadCloned reads the marketplaces.json file tracking cloned repos.
func (s *Store) loadCloned() (map[string]Marketplace, error) {
	data, err := os.ReadFile(s.installedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Marketplace), nil
		}
		return nil, err
	}
	var list []Marketplace
	if err := json.Unmarshal(data, &list); err != nil {
		// Handle legacy null content
		return make(map[string]Marketplace), nil
	}
	m := make(map[string]Marketplace, len(list))
	for _, mp := range list {
		m[mp.Name] = mp
	}
	return m, nil
}

func (s *Store) saveCloned(marketplaces map[string]Marketplace) error {
	dir := filepath.Dir(s.installedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	list := make([]Marketplace, 0, len(marketplaces))
	for _, mp := range marketplaces {
		list = append(list, mp)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.installedPath, data, 0644)
}

// RecordClone saves a marketplace to the cloned records.
func (s *Store) RecordClone(mp Marketplace) error {
	records, err := s.loadCloned()
	if err != nil {
		return err
	}
	mp.ClonedAt = time.Now().Format(time.RFC3339)
	mp.Status = "cloned"
	records[mp.Name] = mp
	return s.saveCloned(records)
}

// RemoveRecord removes a marketplace from the cloned records.
func (s *Store) RemoveRecord(name string) error {
	records, err := s.loadCloned()
	if err != nil {
		return err
	}
	delete(records, name)
	return s.saveCloned(records)
}

// ScanMarketplaces scans pluginsDir for marketplace directories
// and includes recorded marketplaces whose directories may be missing.
func (s *Store) ScanMarketplaces() ([]Marketplace, error) {
	records, _ := s.loadCloned()

	entries, err := os.ReadDir(s.pluginsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	seen := make(map[string]bool)
	var marketplaces []Marketplace
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(s.pluginsDir, entry.Name())
		mp := parseMarketplaceDir(dir)

		// Enrich from saved records (repo URL, clonedAt)
		if rec, ok := records[mp.Name]; ok {
			if mp.RepoURL == "" {
				mp.RepoURL = rec.RepoURL
			}
			mp.ClonedAt = rec.ClonedAt
		}
		mp.Status = "cloned"
		marketplaces = append(marketplaces, mp)
		seen[mp.Name] = true
	}

	// Include recorded marketplaces whose directories are missing on disk.
	// This can happen if the clone was deleted or on case-insensitive FS issues.
	for name, rec := range records {
		if seen[name] {
			continue
		}
		rec.Status = "missing"
		marketplaces = append(marketplaces, rec)
	}

	return marketplaces, nil
}

// AddByRepo clones a GitHub repo and records it as a marketplace.
func (s *Store) AddByRepo(ctx context.Context, repoRef string) (*Marketplace, error) {
	repoURL := repoRef
	if !strings.HasPrefix(repoRef, "http") {
		repoURL = "https://github.com/" + repoRef
	}

	parts := strings.Split(strings.TrimSuffix(repoURL, "/"), "/")
	repoName := parts[len(parts)-1]
	if idx := strings.Index(repoName, ".git"); idx > 0 {
		repoName = repoName[:idx]
	}

	targetDir := filepath.Join(s.pluginsDir, repoName)
	if err := CloneRepo(ctx, repoURL, targetDir); err != nil {
		return nil, fmt.Errorf("cloning %s: %w", repoRef, err)
	}

	mp := parseMarketplaceDir(targetDir)
	if mp.RepoURL == "" {
		mp.RepoURL = repoURL
	}
	mp.Status = "cloned"

	// Rename directory to match the marketplace name from manifest if different.
	// This avoids case mismatches (e.g. URL gives "ECC" but manifest says "ecc").
	if mp.Name != "" && mp.Name != repoName {
		canonicalDir := filepath.Join(s.pluginsDir, mp.Name)
		if canonicalDir != targetDir {
			if strings.EqualFold(mp.Name, repoName) {
				// Case-only difference: on case-insensitive FS (macOS APFS),
				// RemoveAll would delete the source. Rename directly to change
				// the display case, using a temp intermediate to be safe.
				tmpDir := targetDir + ".tmp-rename"
				if err := os.Rename(targetDir, tmpDir); err == nil {
					if err := os.Rename(tmpDir, canonicalDir); err == nil {
						targetDir = canonicalDir
					} else {
						os.Rename(tmpDir, targetDir) // rollback
					}
				}
			} else {
				// Completely different names: safe to remove target first.
				os.RemoveAll(canonicalDir)
				if err := os.Rename(targetDir, canonicalDir); err == nil {
					targetDir = canonicalDir
				}
			}
		}
	}

	if err := s.RecordClone(mp); err != nil {
		return nil, err
	}
	return &mp, nil
}

// RemoveMarketplace deletes the marketplace directory and removes its record.
func (s *Store) RemoveMarketplace(name string) error {
	dir := s.PluginDir(name)
	os.RemoveAll(dir)
	return s.RemoveRecord(name)
}

// parseMarketplaceDir parses a directory into a Marketplace.
// It tries multiple manifest formats in order of priority.
func parseMarketplaceDir(dir string) Marketplace {
	base := filepath.Base(dir)

	// 1. Try .claude-plugin/marketplace.json (Claude Code marketplace format)
	if mp := parseClaudeMarketplace(dir); mp != nil {
		return *mp
	}

	// 2. Try .claude-plugin/plugin.json (Claude Code single plugin format)
	if mp := parseClaudePlugin(dir); mp != nil {
		return *mp
	}

	// 3. Try plugin.json (our custom format)
	if mp := parseCustomManifest(dir); mp != nil {
		return *mp
	}

	// 4. Auto-scan from directory structure
	return autoScanMarketplace(dir, base)
}

// parseClaudeMarketplace parses .claude-plugin/marketplace.json
func parseClaudeMarketplace(dir string) *Marketplace {
	data, err := os.ReadFile(filepath.Join(dir, ".claude-plugin", "marketplace.json"))
	if err != nil {
		return nil
	}

	// Use raw JSON to handle source as both string and object
	var raw struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Author      struct {
			Name string `json:"name"`
		} `json:"owner"`
		Metadata struct {
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"metadata"`
		Plugins []json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	desc := raw.Metadata.Description
	if desc == "" {
		desc = raw.Description
	}

	mp := &Marketplace{
		Name:        raw.Name,
		Description: desc,
		Author:      raw.Author.Name,
		Version:     raw.Metadata.Version,
	}

	for _, rawPlugin := range raw.Plugins {
		item := parseMarketplacePlugin(dir, rawPlugin)
		mp.Plugins = append(mp.Plugins, item)
	}

	// If marketplace has no version, try to use the version from the first plugin
	if mp.Version == "" && len(mp.Plugins) > 0 {
		for _, rawPlugin := range raw.Plugins {
			var pv struct {
				Version string `json:"version"`
			}
			if json.Unmarshal(rawPlugin, &pv) == nil && pv.Version != "" {
				mp.Version = pv.Version
				break
			}
		}
	}

	if len(mp.Plugins) == 0 {
		scanForPlugins(dir, mp)
	}

	return mp
}

// parseMarketplacePlugin handles source as either a string path or an object with URL.
func parseMarketplacePlugin(marketplaceDir string, raw json.RawMessage) PluginItem {
	// Extract common fields first (always strings, won't fail on source type mismatch)
	var base struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Category    string `json:"category"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return PluginItem{Name: "unknown"}
	}

	item := PluginItem{
		Name:        base.Name,
		Description: base.Description,
	}

	// Determine source: could be a string path or an object with URL
	var withSource struct {
		Source json.RawMessage `json:"source"`
	}
	if json.Unmarshal(raw, &withSource) != nil || len(withSource.Source) == 0 {
		// No source field, default to current dir
		item.Path = "."
		pluginSubDir := marketplaceDir
		scanPluginItems(pluginSubDir, &item)
		return item
	}

	// Try string first
	var sourceStr string
	if json.Unmarshal(withSource.Source, &sourceStr) == nil {
		item.Path = sourceStr
		pluginSubDir := filepath.Join(marketplaceDir, strings.TrimPrefix(sourceStr, "./"))
		scanPluginItems(pluginSubDir, &item)
		return item
	}

	// Must be an object with URL
	var sourceObj struct {
		Source string `json:"source"`
		URL    string `json:"url"`
		Ref    string `json:"ref"`
	}
	if json.Unmarshal(withSource.Source, &sourceObj) == nil && sourceObj.URL != "" {
		item.SourceURL = sourceObj.URL
	}
	// External URL plugins don't have local skills to scan

	return item
}

// parseClaudePlugin parses .claude-plugin/plugin.json (single plugin)
func parseClaudePlugin(dir string) *Marketplace {
	data, err := os.ReadFile(filepath.Join(dir, ".claude-plugin", "plugin.json"))
	if err != nil {
		return nil
	}

	var raw struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Author      struct {
			Name string `json:"name"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	base := filepath.Base(dir)
	if raw.Name == "" {
		raw.Name = base
	}

	item := PluginItem{
		Name:        raw.Name,
		Description: raw.Description,
		Path:        ".",
	}
	scanPluginItems(dir, &item)

	return &Marketplace{
		Name:        raw.Name,
		Description: raw.Description,
		Version:     raw.Version,
		Author:      raw.Author.Name,
		Plugins:     []PluginItem{item},
	}
}

// parseCustomManifest parses our custom plugin.json format
func parseCustomManifest(dir string) *Marketplace {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return nil
	}

	var raw struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Author      string   `json:"author"`
		Tags        []string `json:"tags"`
		Skills      []string `json:"skills"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	item := PluginItem{
		Name:        raw.Name,
		Description: raw.Description,
		Path:        ".",
	}
	scanPluginItems(dir, &item)

	return &Marketplace{
		Name:        raw.Name,
		Description: raw.Description,
		Version:     raw.Version,
		Author:      raw.Author,
		Tags:        raw.Tags,
		Plugins:     []PluginItem{item},
	}
}

// autoScanMarketplace creates a Marketplace from directory structure alone
func autoScanMarketplace(dir, baseName string) Marketplace {
	item := PluginItem{
		Name:        baseName,
		Description: "Plugin from " + baseName,
		Path:        ".",
	}
	scanPluginItems(dir, &item)

	return Marketplace{
		Name:        baseName,
		Description: item.Description,
		Version:     "0.0.1",
		Plugins:     []PluginItem{item},
	}
}

// scanPluginItems discovers commands and skills in a plugin directory
func scanPluginItems(dir string, item *PluginItem) {
	// Scan commands/
	cmdsDir := filepath.Join(dir, "commands")
	if entries, err := os.ReadDir(cmdsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				item.Commands = append(item.Commands, name)
			}
		}
	}

	// Scan skills/
	skillsDir := filepath.Join(dir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				item.Skills = append(item.Skills, e.Name())
			}
		}
	}
}

// scanForPlugins scans for plugin subdirectories inside a marketplace
func scanForPlugins(dir string, mp *Marketplace) {
	// Check for plugins/ subdirectory (Claude Code convention)
	pluginsDir := filepath.Join(dir, "plugins")
	if entries, err := os.ReadDir(pluginsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			pluginPath := filepath.Join(pluginsDir, e.Name())
			item := PluginItem{
				Name: e.Name(),
				Path: "plugins/" + e.Name(),
			}
			// Try to read .claude-plugin/plugin.json for description
			descData, err := os.ReadFile(filepath.Join(pluginPath, ".claude-plugin", "plugin.json"))
			if err == nil {
				var desc struct {
					Description string `json:"description"`
				}
				if json.Unmarshal(descData, &desc) == nil {
					item.Description = desc.Description
				}
			}
			scanPluginItems(pluginPath, &item)
			mp.Plugins = append(mp.Plugins, item)
		}
	}
}

// RegistryClient fetches available marketplaces from a remote registry.
type RegistryClient struct {
	httpClient  *http.Client
	registryURL string
}

func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		registryURL: defaultRegistryURL,
	}
}

func (c *RegistryClient) FetchAvailable(ctx context.Context) ([]Marketplace, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.registryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// The registry stores Plugin items — convert to Marketplace
	var plugins []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
		Author      string `json:"author"`
		RepoURL     string `json:"repo_url"`
	}
	if err := json.Unmarshal(data, &plugins); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	var marketplaces []Marketplace
	for _, p := range plugins {
		marketplaces = append(marketplaces, Marketplace{
			Name:        p.Name,
			Description: p.Description,
			Version:     p.Version,
			Author:      p.Author,
			RepoURL:     p.RepoURL,
			Status:      "available",
		})
	}
	return marketplaces, nil
}

// MergeMarketplaces merges local cloned marketplaces with remote available ones.
func MergeMarketplaces(local, remote []Marketplace) []Marketplace {
	seen := make(map[string]bool)
	merged := make([]Marketplace, 0, len(local)+len(remote))

	for _, mp := range local {
		seen[mp.Name] = true
		merged = append(merged, mp)
	}

	for _, mp := range remote {
		if !seen[mp.Name] {
			merged = append(merged, mp)
		}
	}

	return merged
}

// CloneRepo performs a git clone --depth 1 of the given repo.
func CloneRepo(ctx context.Context, repoURL, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	if _, err := os.Stat(targetDir); err == nil {
		os.RemoveAll(targetDir)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, targetDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}
	return nil
}

// ResolvePluginDir returns the local directory for a plugin, cloning from SourceURL if needed.
// If the plugin has an external SourceURL, it clones to store's cache and returns that path.
// Otherwise it returns the local path within sourceDir.
func (s *Store) ResolvePluginDir(sourceDir string, pi PluginItem) (string, error) {
	if pi.SourceURL != "" {
		// External plugin: clone to cache/<marketplace>/<plugin>
		repoName := repoNameFromURL(pi.SourceURL)
		cacheDir := filepath.Join(s.pluginsDir, ".cache", repoName)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		if err := CloneRepo(ctx, pi.SourceURL, cacheDir); err != nil {
			return "", fmt.Errorf("cloning external plugin %s: %w", pi.Name, err)
		}
		return cacheDir, nil
	}

	relPath := strings.TrimPrefix(pi.Path, "./")
	if relPath == "" || relPath == "." {
		return sourceDir, nil
	}
	return filepath.Join(sourceDir, relPath), nil
}

// repoNameFromURL extracts the repo name from a git URL.
func repoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return url
}
