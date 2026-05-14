package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultRegistryURL = "https://raw.githubusercontent.com/ying-sun1/skill-tui/main/configs/registry.json"

type Client struct {
	httpClient   *http.Client
	registryURL  string
	localPaths   []string
	cache        *Cache
}

type Publisher struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RepoURL     string `json:"repo_url"`
	SkillCount  int    `json:"skill_count"`
}

type MarketSkill struct {
	Name        string   `json:"name"`
	PublisherID string   `json:"publisher_id"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Path        string   `json:"path"`
	RepoURL     string   `json:"repo_url"`
}

type Registry struct {
	Publishers []Publisher   `json:"publishers"`
	Skills     []MarketSkill `json:"skills"`
	UpdatedAt  string        `json:"updated_at"`
}

func NewClient() *Client {
	home, _ := os.UserHomeDir()
	exe, _ := os.Executable()

	localPaths := []string{
		filepath.Join(home, ".skill-tui", "registry.json"),
	}

	if exe != "" {
		localPaths = append(localPaths, filepath.Join(filepath.Dir(exe), "configs", "registry.json"))
	}
	localPaths = append(localPaths, "configs/registry.json")

	return &Client{
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		registryURL:  defaultRegistryURL,
		localPaths:   localPaths,
		cache:        NewCache(),
	}
}

func (c *Client) WithRegistryURL(url string) *Client {
	c.registryURL = url
	return c
}

func (c *Client) FetchRegistry(ctx context.Context) (*Registry, error) {
	if cached := c.cache.Get("registry"); cached != nil {
		if reg, ok := cached.(*Registry); ok {
			return reg, nil
		}
	}

	reg, err := c.fetchRemote(ctx)
	if err == nil {
		c.cache.Set("registry", reg)
		return reg, nil
	}

	reg, localErr := c.fetchLocal()
	if localErr == nil {
		c.cache.Set("registry", reg)
		return reg, nil
	}

	return nil, fmt.Errorf("remote: %v; local: %v", err, localErr)
}

func (c *Client) fetchRemote(ctx context.Context) (*Registry, error) {
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

	return parseRegistry(data)
}

func (c *Client) fetchLocal() (*Registry, error) {
	for _, p := range c.localPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		reg, err := parseRegistry(data)
		if err != nil {
			continue
		}
		return reg, nil
	}
	return nil, fmt.Errorf("no local registry found")
}

func parseRegistry(data []byte) (*Registry, error) {
	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	return &registry, nil
}

func (c *Client) ListPublishers(ctx context.Context) ([]Publisher, error) {
	reg, err := c.FetchRegistry(ctx)
	if err != nil {
		return nil, err
	}

	for i := range reg.Publishers {
		reg.Publishers[i].SkillCount = countSkills(reg.Skills, reg.Publishers[i].ID)
	}
	return reg.Publishers, nil
}

func (c *Client) ListSkills(ctx context.Context, publisherID string) ([]MarketSkill, error) {
	reg, err := c.FetchRegistry(ctx)
	if err != nil {
		return nil, err
	}

	if publisherID == "" {
		return reg.Skills, nil
	}

	var filtered []MarketSkill
	for _, s := range reg.Skills {
		if s.PublisherID == publisherID {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

func (c *Client) Search(ctx context.Context, query string) ([]MarketSkill, error) {
	reg, err := c.FetchRegistry(ctx)
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var results []MarketSkill
	for _, s := range reg.Skills {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Description), q) ||
			containsTag(s.Tags, q) {
			results = append(results, s)
		}
	}
	return results, nil
}

func countSkills(skills []MarketSkill, publisherID string) int {
	count := 0
	for _, s := range skills {
		if s.PublisherID == publisherID {
			count++
		}
	}
	return count
}

func containsTag(tags []string, q string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}
