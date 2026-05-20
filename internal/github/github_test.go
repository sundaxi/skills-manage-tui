package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-token")
	if c.token != "test-token" {
		t.Errorf("token = %q", c.token)
	}
	if c.baseURL != "https://api.github.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestWithToken(t *testing.T) {
	c := NewClient("old")
	nc := c.WithToken("new")
	if nc.token != "new" {
		t.Errorf("new token = %q", nc.token)
	}
	if c.token != "old" {
		t.Error("original client should be unchanged")
	}
}

func TestParseRepoURL_HTTPS(t *testing.T) {
	c := NewClient("")
	tests := []struct {
		url   string
		owner string
		name  string
	}{
		{"https://github.com/owner/repo", "owner", "repo"},
		{"https://github.com/owner/repo/", "owner", "repo"},
		{"https://github.com/owner/repo.git", "owner", "repo"},
		{"https://github.com/some-org/my-repo.git/", "some-org", "my-repo"},
	}

	for _, tt := range tests {
		info, err := c.ParseRepoURL(tt.url)
		if err != nil {
			t.Errorf("ParseRepoURL(%q) error: %v", tt.url, err)
			continue
		}
		if info.Owner != tt.owner {
			t.Errorf("ParseRepoURL(%q).Owner = %q, want %q", tt.url, info.Owner, tt.owner)
		}
		if info.Name != tt.name {
			t.Errorf("ParseRepoURL(%q).Name = %q, want %q", tt.url, info.Name, tt.name)
		}
	}
}

func TestParseRepoURL_Invalid(t *testing.T) {
	c := NewClient("")
	invalid := []string{
		"not-a-url",
		"https://gitlab.com/owner/repo",
		"https://github.com/owner-only",
	}
	for _, url := range invalid {
		_, err := c.ParseRepoURL(url)
		if err == nil {
			t.Errorf("ParseRepoURL(%q) should error", url)
		}
	}
}

func TestGetRepo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Error("missing Accept header")
		}
		json.NewEncoder(w).Encode(map[string]string{
			"full_name":      "owner/repo",
			"default_branch": "main",
		})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	info, err := c.GetRepo(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepo error: %v", err)
	}
	if info.FullName != "owner/repo" {
		t.Errorf("FullName = %q", info.FullName)
	}
	if info.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q", info.DefaultBranch)
	}
}

func TestGetTree(t *testing.T) {
	tree := []TreeEntry{
		{Path: "skills/test/SKILL.md", Type: "blob"},
		{Path: "README.md", Type: "blob"},
		{Path: "skills", Type: "tree"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": tree,
		})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	entries, err := c.GetTree(context.Background(), "owner", "repo", "main")
	if err != nil {
		t.Fatalf("GetTree error: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("entries = %d, want 3", len(entries))
	}
}

func TestGetTree_DefaultRef(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/git/trees/HEAD" {
			t.Errorf("expected HEAD ref, got path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"tree": []TreeEntry{}})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	_, err := c.GetTree(context.Background(), "o", "r", "")
	if err != nil {
		t.Fatalf("GetTree error: %v", err)
	}
}

func TestGetFileContent_WithDownloadURL(t *testing.T) {
	content := "# Hello World"
	var serverURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/o/r/contents/README.md" {
			json.NewEncoder(w).Encode(FileContent{
				Name:        "README.md",
				DownloadURL: serverURL + "/raw/README.md",
			})
			return
		}
		if r.URL.Path == "/raw/README.md" {
			w.Write([]byte(content))
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()
	serverURL = ts.URL

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	got, err := c.GetFileContent(context.Background(), "o", "r", "README.md", "")
	if err != nil {
		t.Fatalf("GetFileContent error: %v", err)
	}
	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestGetFileContent_WithRef(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "v1.0" {
			t.Errorf("expected ref=v1.0, got %q", r.URL.Query().Get("ref"))
		}
		json.NewEncoder(w).Encode(FileContent{Content: "inline content"})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	got, err := c.GetFileContent(context.Background(), "o", "r", "file.md", "v1.0")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got != "inline content" {
		t.Errorf("content = %q", got)
	}
}

func TestListRepos(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repos := []map[string]string{
			{"full_name": "user/repo1", "name": "repo1", "default_branch": "main"},
			{"full_name": "user/repo2", "name": "repo2", "default_branch": "master"},
		}
		json.NewEncoder(w).Encode(repos)
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	repos, err := c.ListRepos(context.Background(), "user")
	if err != nil {
		t.Fatalf("ListRepos error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("repos = %d, want 2", len(repos))
	}
	if repos[0].Name != "repo1" {
		t.Errorf("repos[0].Name = %q", repos[0].Name)
	}
}

func TestGet_WithAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token" {
			t.Errorf("Authorization = %q, want Bearer my-token", auth)
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client(), token: "my-token"}
	var result map[string]string
	err := c.get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
}

func TestGet_NoAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("should have no auth header, got %q", auth)
		}
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	var result map[string]string
	err := c.get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
}

func TestGet_Forbidden(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("rate limited"))
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	var result interface{}
	err := c.get(context.Background(), "/test", &result)
	if err == nil {
		t.Error("expected error for 403")
	}
}

func TestGet_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	var result interface{}
	err := c.get(context.Background(), "/test", &result)
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestGet_NilResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	err := c.get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("get with nil result should succeed: %v", err)
	}
}

func TestExtractSkillName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"skills/my-skill/SKILL.md", "my"},           // strips -skill suffix
		{"skills/skill-coding/SKILL.md", "coding"},   // strips skill- prefix
		{"skills/testing-skill/SKILL.md", "testing"}, // strips -skill suffix
		{"deep/path/to/skill-name/SKILL.md", "name"}, // strips skill- prefix
		{"SKILL.md", "."},
	}

	for _, tt := range tests {
		entry := TreeEntry{Path: tt.path}
		got := extractSkillName(entry)
		if got != tt.want {
			t.Errorf("extractSkillName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFindSkillFiles(t *testing.T) {
	imp := NewImporter(nil, "")
	tree := []TreeEntry{
		{Path: "skills/a/SKILL.md", Type: "blob"},
		{Path: "skills/b/SKILL.md", Type: "blob"},
		{Path: "README.md", Type: "blob"},
		{Path: "skills/a/skill.md", Type: "blob"},
		{Path: "skills", Type: "tree"},
		{Path: "other/c/SKILL.md", Type: "blob"},
	}

	// No subPath filter — SKILL.md and skill.md both match (case insensitive)
	matches := imp.findSkillFiles(tree, "")
	if len(matches) != 4 {
		t.Errorf("no filter: got %d matches, want 4", len(matches))
	}

	// With subPath filter — skills/a has SKILL.md + skill.md = 2
	matches = imp.findSkillFiles(tree, "skills/a")
	if len(matches) != 2 {
		t.Errorf("subPath=skills/a: got %d, want 2", len(matches))
	}

	// subPath that matches nothing
	matches = imp.findSkillFiles(tree, "nonexistent")
	if len(matches) != 0 {
		t.Errorf("nonexistent subPath: got %d, want 0", len(matches))
	}
}

func TestImportFromURL(t *testing.T) {
	skillContent := "# Test Skill\nHello"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/owner/repo":
			json.NewEncoder(w).Encode(map[string]string{
				"full_name":      "owner/repo",
				"default_branch": "main",
			})
		case r.URL.Path == "/repos/owner/repo/git/trees/main":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tree": []TreeEntry{
					{Path: "skills/test/SKILL.md", Type: "blob"},
				},
			})
		case r.URL.Path == "/repos/owner/repo/contents/skills/test/SKILL.md":
			json.NewEncoder(w).Encode(FileContent{
				Content:     skillContent,
				DownloadURL: "",
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	skillsDir := filepath.Join(t.TempDir(), "skills")
	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	imp := NewImporter(c, skillsDir)

	results, err := imp.ImportFromURL(context.Background(), "https://github.com/owner/repo", "")
	if err != nil {
		t.Fatalf("ImportFromURL error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Skipped {
		t.Error("should not be skipped")
	}
	if results[0].SkillName != "test" {
		t.Errorf("SkillName = %q, want test", results[0].SkillName)
	}

	// Verify file was written
	written, err := os.ReadFile(filepath.Join(skillsDir, "test", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(written) != skillContent {
		t.Errorf("written content = %q, want %q", string(written), skillContent)
	}
}

func TestImportFromRepo_NoSkillFiles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tree": []TreeEntry{
				{Path: "README.md", Type: "blob"},
			},
		})
	}))
	defer ts.Close()

	c := &Client{baseURL: ts.URL, httpClient: ts.Client()}
	imp := NewImporter(c, t.TempDir())

	repo := &RepoInfo{Owner: "o", Name: "r", DefaultBranch: "main", FullName: "o/r"}
	_, err := imp.ImportFromRepo(context.Background(), repo, "")
	if err == nil {
		t.Error("expected error for no SKILL.md files")
	}
}

func TestNewImporter(t *testing.T) {
	c := NewClient("token")
	imp := NewImporter(c, "/tmp/skills")
	if imp.skillsDir != "/tmp/skills" {
		t.Errorf("skillsDir = %q", imp.skillsDir)
	}
	if imp.client != c {
		t.Error("client mismatch")
	}
}
