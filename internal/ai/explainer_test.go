package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewExplainer_OpenAI(t *testing.T) {
	e := NewExplainer("openai", "key123", "")
	if e.provider != "openai" {
		t.Errorf("provider = %q", e.provider)
	}
	if e.model != "gpt-4o-mini" {
		t.Errorf("model = %q", e.model)
	}
	if e.endpoint != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("endpoint = %q", e.endpoint)
	}
	if e.apiKey != "key123" {
		t.Errorf("apiKey = %q", e.apiKey)
	}
}

func TestNewExplainer_Anthropic(t *testing.T) {
	e := NewExplainer("anthropic", "key456", "")
	if e.model != "claude-haiku-4-5-20251001" {
		t.Errorf("model = %q", e.model)
	}
	if e.endpoint != "https://api.anthropic.com/v1/messages" {
		t.Errorf("endpoint = %q", e.endpoint)
	}
}

func TestNewExplainer_CustomEndpoint(t *testing.T) {
	e := NewExplainer("openai", "key", "http://custom.example.com/v1")
	if e.endpoint != "http://custom.example.com/v1" {
		t.Errorf("endpoint = %q", e.endpoint)
	}
}

func TestTruncateContent(t *testing.T) {
	short := "hello"
	if got := truncateContent(short, 100); got != "hello" {
		t.Errorf("short content truncated: %q", got)
	}

	long := strings.Repeat("x", 200)
	got := truncateContent(long, 100)
	if len(got) <= 100 {
		t.Errorf("should include truncation marker")
	}
	if !strings.Contains(got, "... (truncated)") {
		t.Errorf("missing truncation marker: %q", got)
	}
}

func TestExplain_NoAPIKey(t *testing.T) {
	e := NewExplainer("openai", "", "")
	_, err := e.Explain(context.Background(), "test", "content")
	if err == nil {
		t.Error("expected error for no API key")
	}
}

func TestExplain_CacheHit(t *testing.T) {
	dir := t.TempDir()
	e := &Explainer{
		provider: "openai",
		apiKey:   "test-key",
		cacheDir: dir,
	}

	exp := Explanation{SkillName: "test", Content: "cached explanation"}
	data, _ := json.MarshalIndent(exp, "", "  ")
	os.WriteFile(filepath.Join(dir, "test.json"), data, 0644)

	result, err := e.Explain(context.Background(), "test", "content")
	if err != nil {
		t.Fatalf("Explain error: %v", err)
	}
	if result != "cached explanation" {
		t.Errorf("result = %q, want cached explanation", result)
	}
}

func TestCallOpenAI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Explained!"}},
			},
		})
	}))
	defer ts.Close()

	e := &Explainer{
		apiKey:   "test-key",
		endpoint: ts.URL,
		model:    "gpt-test",
		client:   ts.Client(),
	}

	result, err := e.callOpenAI(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("callOpenAI error: %v", err)
	}
	if result != "Explained!" {
		t.Errorf("result = %q", result)
	}
}

func TestCallOpenAI_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "rate limited"},
		})
	}))
	defer ts.Close()

	e := &Explainer{apiKey: "key", endpoint: ts.URL, model: "test", client: ts.Client()}
	_, err := e.callOpenAI(context.Background(), "test")
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestCallOpenAI_NoChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{}})
	}))
	defer ts.Close()

	e := &Explainer{apiKey: "key", endpoint: ts.URL, model: "test", client: ts.Client()}
	_, err := e.callOpenAI(context.Background(), "test")
	if err == nil {
		t.Error("expected error for no choices")
	}
}

func TestCallAnthropic(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("version = %q", r.Header.Get("anthropic-version"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"text": "Anthropic says hello"}},
		})
	}))
	defer ts.Close()

	e := &Explainer{apiKey: "test-key", endpoint: ts.URL, model: "claude-test", client: ts.Client()}
	result, err := e.callAnthropic(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("callAnthropic error: %v", err)
	}
	if result != "Anthropic says hello" {
		t.Errorf("result = %q", result)
	}
}

func TestCallAnthropic_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "invalid key"},
		})
	}))
	defer ts.Close()

	e := &Explainer{apiKey: "key", endpoint: ts.URL, model: "test", client: ts.Client()}
	_, err := e.callAnthropic(context.Background(), "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCallAnthropic_NoContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"content": []interface{}{}})
	}))
	defer ts.Close()

	e := &Explainer{apiKey: "key", endpoint: ts.URL, model: "test", client: ts.Client()}
	_, err := e.callAnthropic(context.Background(), "test")
	if err == nil {
		t.Error("expected error for no content")
	}
}

func TestLoadCache_Missing(t *testing.T) {
	e := &Explainer{cacheDir: t.TempDir()}
	_, err := e.loadCache("missing")
	if err == nil {
		t.Error("expected error for missing cache")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	dir := t.TempDir()
	e := &Explainer{cacheDir: dir, provider: "test"}

	err := e.saveCache("my-skill", "explanation text")
	if err != nil {
		t.Fatalf("saveCache error: %v", err)
	}

	content, err := e.loadCache("my-skill")
	if err != nil {
		t.Fatalf("loadCache error: %v", err)
	}
	if content != "explanation text" {
		t.Errorf("content = %q", content)
	}
}

func TestExplain_OpenAIIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "This skill does X"}},
			},
		})
	}))
	defer ts.Close()

	e := &Explainer{
		provider: "openai",
		apiKey:   "test-key",
		endpoint: ts.URL,
		model:    "test",
		client:   ts.Client(),
		cacheDir: t.TempDir(),
	}

	result, err := e.Explain(context.Background(), "test-skill", "# Test\nDoes things")
	if err != nil {
		t.Fatalf("Explain error: %v", err)
	}
	if result != "This skill does X" {
		t.Errorf("result = %q", result)
	}

	// Second call should hit cache
	result2, err := e.Explain(context.Background(), "test-skill", "different content")
	if err != nil {
		t.Fatalf("cached Explain error: %v", err)
	}
	if result2 != result {
		t.Errorf("cached result = %q, want %q", result2, result)
	}
}

func TestExplain_AnthropicIntegration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"text": "Anthropic explanation"}},
		})
	}))
	defer ts.Close()

	e := &Explainer{
		provider: "anthropic",
		apiKey:   "test-key",
		endpoint: ts.URL,
		model:    "test",
		client:   ts.Client(),
		cacheDir: t.TempDir(),
	}

	result, err := e.Explain(context.Background(), "test-skill", "content")
	if err != nil {
		t.Fatalf("Explain error: %v", err)
	}
	if result != "Anthropic explanation" {
		t.Errorf("result = %q", result)
	}
}
