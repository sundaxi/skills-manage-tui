package ai

import (
	"bytes"
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

type Explainer struct {
	provider string
	apiKey   string
	endpoint string
	model    string
	client   *http.Client
	cacheDir string
}

type Explanation struct {
	SkillName   string `json:"skill_name"`
	Content     string `json:"content"`
	Provider    string `json:"provider"`
	GeneratedAt string `json:"generated_at"`
}

func NewExplainer(provider, apiKey, endpoint string) *Explainer {
	model := "gpt-4o-mini"
	if provider == "anthropic" {
		model = "claude-haiku-4-5-20251001"
	}

	if endpoint == "" {
		switch provider {
		case "openai":
			endpoint = "https://api.openai.com/v1/chat/completions"
		case "anthropic":
			endpoint = "https://api.anthropic.com/v1/messages"
		}
	}

	home, _ := os.UserHomeDir()

	return &Explainer{
		provider: provider,
		apiKey:   apiKey,
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{Timeout: 60 * time.Second},
		cacheDir: filepath.Join(home, ".skill-tui", "cache"),
	}
}

func (e *Explainer) Explain(ctx context.Context, skillName, skillContent string) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("AI API key not configured. Run: skill-tui config set ai_key <key>")
	}

	if cached, err := e.loadCache(skillName); err == nil && cached != "" {
		return cached, nil
	}

	prompt := fmt.Sprintf(`Explain the following AI coding agent skill in clear, concise language.
Describe what it does, when to use it, and key features. Keep the explanation under 200 words.
Respond in the same language as the skill content.

Skill name: %s

Skill content:
%s`, skillName, truncateContent(skillContent, 3000))

	var result string
	var err error

	switch e.provider {
	case "openai":
		result, err = e.callOpenAI(ctx, prompt)
	case "anthropic":
		result, err = e.callAnthropic(ctx, prompt)
	default:
		result, err = e.callOpenAI(ctx, prompt)
	}

	if err != nil {
		return "", err
	}

	_ = e.saveCache(skillName, result)
	return result, nil
}

func (e *Explainer) callOpenAI(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"model": e.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 500,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func (e *Explainer) callAnthropic(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"model":      e.model,
		"max_tokens": 500,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return strings.TrimSpace(result.Content[0].Text), nil
}

func (e *Explainer) loadCache(skillName string) (string, error) {
	path := filepath.Join(e.cacheDir, skillName+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var exp Explanation
	if err := json.Unmarshal(data, &exp); err != nil {
		return "", err
	}
	return exp.Content, nil
}

func (e *Explainer) saveCache(skillName, content string) error {
	if err := os.MkdirAll(e.cacheDir, 0755); err != nil {
		return err
	}
	exp := Explanation{
		SkillName:   skillName,
		Content:     content,
		Provider:    e.provider,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(exp, "", "  ")
	return os.WriteFile(filepath.Join(e.cacheDir, skillName+".json"), data, 0644)
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... (truncated)"
}
