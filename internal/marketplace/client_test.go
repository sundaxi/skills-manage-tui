package marketplace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseRegistry_Valid(t *testing.T) {
	data := `{
		"publishers": [{"id": "pub1", "name": "Publisher One"}],
		"skills": [{"name": "skill-a", "publisher_id": "pub1", "description": "A skill"}],
		"updated_at": "2024-01-01"
	}`
	reg, err := parseRegistry([]byte(data))
	if err != nil {
		t.Fatalf("parseRegistry error: %v", err)
	}
	if len(reg.Publishers) != 1 {
		t.Errorf("publishers count = %d, want 1", len(reg.Publishers))
	}
	if len(reg.Skills) != 1 {
		t.Errorf("skills count = %d, want 1", len(reg.Skills))
	}
	if reg.UpdatedAt != "2024-01-01" {
		t.Errorf("updated_at = %q", reg.UpdatedAt)
	}
}

func TestParseRegistry_Invalid(t *testing.T) {
	_, err := parseRegistry([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseRegistry_Empty(t *testing.T) {
	reg, err := parseRegistry([]byte("{}"))
	if err != nil {
		t.Fatalf("parseRegistry error: %v", err)
	}
	if len(reg.Publishers) != 0 {
		t.Errorf("publishers = %d, want 0", len(reg.Publishers))
	}
}

func TestCountSkills(t *testing.T) {
	skills := []MarketSkill{
		{Name: "a", PublisherID: "p1"},
		{Name: "b", PublisherID: "p1"},
		{Name: "c", PublisherID: "p2"},
	}
	if got := countSkills(skills, "p1"); got != 2 {
		t.Errorf("countSkills(p1) = %d, want 2", got)
	}
	if got := countSkills(skills, "p2"); got != 1 {
		t.Errorf("countSkills(p2) = %d, want 1", got)
	}
	if got := countSkills(skills, "p3"); got != 0 {
		t.Errorf("countSkills(p3) = %d, want 0", got)
	}
}

func TestContainsTag(t *testing.T) {
	tags := []string{"Go", "Testing", "CI/CD"}

	if !containsTag(tags, "go") {
		t.Error("should find 'go' (case insensitive)")
	}
	if !containsTag(tags, "test") {
		t.Error("should find 'test' (substring)")
	}
	if containsTag(tags, "python") {
		t.Error("should not find 'python'")
	}
	if containsTag(nil, "any") {
		t.Error("nil tags should return false")
	}
}

func testRegistry() *Registry {
	return &Registry{
		Publishers: []Publisher{
			{ID: "pub1", Name: "Publisher One"},
			{ID: "pub2", Name: "Publisher Two"},
		},
		Skills: []MarketSkill{
			{Name: "skill-a", PublisherID: "pub1", Description: "Alpha skill", Tags: []string{"go"}},
			{Name: "skill-b", PublisherID: "pub1", Description: "Beta skill", Tags: []string{"python"}},
			{Name: "skill-c", PublisherID: "pub2", Description: "Gamma skill", Tags: []string{"go", "testing"}},
		},
		UpdatedAt: "2024-01-01",
	}
}

func TestFetchRegistry_Remote(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		registryURL: ts.URL,
		localPaths:  nil,
		cache:       NewCache(),
	}

	got, err := c.FetchRegistry(context.Background())
	if err != nil {
		t.Fatalf("FetchRegistry error: %v", err)
	}
	if len(got.Skills) != 3 {
		t.Errorf("skills = %d, want 3", len(got.Skills))
	}
}

func TestFetchRegistry_CacheHit(t *testing.T) {
	reg := testRegistry()
	c := &Client{
		httpClient:  http.DefaultClient,
		registryURL: "http://invalid.example.com",
		localPaths:  nil,
		cache:       NewCache(),
	}
	c.cache.Set("registry", reg)

	got, err := c.FetchRegistry(context.Background())
	if err != nil {
		t.Fatalf("FetchRegistry error: %v", err)
	}
	if got != reg {
		t.Error("should return cached registry")
	}
}

func TestFetchRegistry_FallbackToLocal(t *testing.T) {
	dir := t.TempDir()
	reg := testRegistry()
	data, _ := json.Marshal(reg)
	localPath := filepath.Join(dir, "registry.json")
	os.WriteFile(localPath, data, 0644)

	c := &Client{
		httpClient:  http.DefaultClient,
		registryURL: "http://invalid.example.com/404",
		localPaths:  []string{localPath},
		cache:       NewCache(),
	}

	got, err := c.FetchRegistry(context.Background())
	if err != nil {
		t.Fatalf("FetchRegistry fallback error: %v", err)
	}
	if len(got.Skills) != 3 {
		t.Errorf("skills = %d, want 3", len(got.Skills))
	}
}

func TestFetchRegistry_NoSourcesAvailable(t *testing.T) {
	c := &Client{
		httpClient:  http.DefaultClient,
		registryURL: "http://invalid.example.com/404",
		localPaths:  []string{"/nonexistent/path"},
		cache:       NewCache(),
	}

	_, err := c.FetchRegistry(context.Background())
	if err == nil {
		t.Error("expected error when no sources available")
	}
}

func TestListPublishers(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		registryURL: ts.URL,
		localPaths:  nil,
		cache:       NewCache(),
	}

	pubs, err := c.ListPublishers(context.Background())
	if err != nil {
		t.Fatalf("ListPublishers error: %v", err)
	}
	if len(pubs) != 2 {
		t.Fatalf("publishers = %d, want 2", len(pubs))
	}
	// pub1 has 2 skills
	for _, p := range pubs {
		if p.ID == "pub1" && p.SkillCount != 2 {
			t.Errorf("pub1 SkillCount = %d, want 2", p.SkillCount)
		}
		if p.ID == "pub2" && p.SkillCount != 1 {
			t.Errorf("pub2 SkillCount = %d, want 1", p.SkillCount)
		}
	}
}

func TestListSkills_All(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	skills, err := c.ListSkills(context.Background(), "")
	if err != nil {
		t.Fatalf("ListSkills error: %v", err)
	}
	if len(skills) != 3 {
		t.Errorf("skills = %d, want 3", len(skills))
	}
}

func TestListSkills_FilterByPublisher(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	skills, err := c.ListSkills(context.Background(), "pub2")
	if err != nil {
		t.Fatalf("ListSkills error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("skills = %d, want 1", len(skills))
	}
	if skills[0].Name != "skill-c" {
		t.Errorf("name = %q, want skill-c", skills[0].Name)
	}
}

func TestSearch_ByName(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	results, err := c.Search(context.Background(), "skill-a")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
}

func TestSearch_ByDescription(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	results, err := c.Search(context.Background(), "gamma")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Name != "skill-c" {
		t.Errorf("name = %q", results[0].Name)
	}
}

func TestSearch_ByTag(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	results, err := c.Search(context.Background(), "python")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Name != "skill-b" {
		t.Errorf("name = %q", results[0].Name)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	results, err := c.Search(context.Background(), "ALPHA")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results = %d, want 1", len(results))
	}
}

func TestSearch_NoResults(t *testing.T) {
	reg := testRegistry()
	data, _ := json.Marshal(reg)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}

	results, err := c.Search(context.Background(), "zzzznonexistent")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
}

func TestWithRegistryURL(t *testing.T) {
	c := &Client{registryURL: "http://old.example.com"}
	c.WithRegistryURL("http://new.example.com")
	if c.registryURL != "http://new.example.com" {
		t.Errorf("registryURL = %q", c.registryURL)
	}
}

func TestFetchRemote_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := &Client{httpClient: ts.Client(), registryURL: ts.URL, cache: NewCache()}
	_, err := c.fetchRemote(context.Background())
	if err == nil {
		t.Error("expected error for HTTP 500")
	}
}

func TestFetchLocal_NotFound(t *testing.T) {
	c := &Client{localPaths: []string{"/nonexistent"}}
	_, err := c.fetchLocal()
	if err == nil {
		t.Error("expected error for no local files")
	}
}

func TestFetchLocal_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	c := &Client{localPaths: []string{path}}
	_, err := c.fetchLocal()
	if err == nil {
		t.Error("expected error for corrupt file")
	}
}

func TestFetchLocal_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reg.json")
	data, _ := json.Marshal(testRegistry())
	os.WriteFile(path, data, 0644)

	c := &Client{localPaths: []string{"/nonexistent", path}}
	reg, err := c.fetchLocal()
	if err != nil {
		t.Fatalf("fetchLocal error: %v", err)
	}
	if len(reg.Skills) != 3 {
		t.Errorf("skills = %d", len(reg.Skills))
	}
}
