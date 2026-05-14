package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

type RepoInfo struct {
	Owner    string
	Name     string
	FullName string
	DefaultBranch string
}

type TreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"` // "blob" or "tree"
	SHA  string `json:"sha"`
}

type FileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) WithToken(token string) *Client {
	nc := *c
	nc.token = token
	return &nc
}

func (c *Client) ParseRepoURL(rawURL string) (*RepoInfo, error) {
	rawURL = strings.TrimSuffix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, ".git")

	parts := strings.Split(rawURL, "github.com/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL: %s", rawURL)
	}
	segments := strings.Split(parts[1], "/")
	if len(segments) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL: %s", rawURL)
	}

	return &RepoInfo{
		Owner: segments[0],
		Name:  segments[1],
		FullName: segments[0] + "/" + segments[1],
	}, nil
}

func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*RepoInfo, error) {
	var result struct {
		FullName string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &result); err != nil {
		return nil, err
	}
	return &RepoInfo{
		Owner:    owner,
		Name:     repo,
		FullName: result.FullName,
		DefaultBranch: result.DefaultBranch,
	}, nil
}

func (c *Client) GetTree(ctx context.Context, owner, repo, ref string) ([]TreeEntry, error) {
	if ref == "" {
		ref = "HEAD"
	}
	var result struct {
		Tree []TreeEntry `json:"tree"`
	}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, ref), &result); err != nil {
		return nil, err
	}
	return result.Tree, nil
}

func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	url := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}

	var fc FileContent
	if err := c.get(ctx, url, &fc); err != nil {
		return "", err
	}

	if fc.DownloadURL != "" {
		return c.downloadRaw(ctx, fc.DownloadURL)
	}

	return fc.Content, nil
}

func (c *Client) ListRepos(ctx context.Context, owner string) ([]RepoInfo, error) {
	var repos []struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := c.get(ctx, fmt.Sprintf("/users/%s/repos?per_page=100", owner), &repos); err != nil {
		return nil, err
	}

	var result []RepoInfo
	for _, r := range repos {
		result = append(result, RepoInfo{
			Owner:    owner,
			Name:     r.Name,
			FullName: r.FullName,
			DefaultBranch: r.DefaultBranch,
		})
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rate limited or forbidden: %s", string(body))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) downloadRaw(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
