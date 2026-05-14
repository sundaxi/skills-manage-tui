package skill

import (
	"bufio"
	"strings"
)

type Metadata struct {
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	Tags        []string `yaml:"tags"`
}

func parseMetadata(content string) Metadata {
	meta := Metadata{}
	fm := extractFrontmatter(content)
	if fm == "" {
		return meta
	}

	lines := strings.Split(fm, "\n")
	for _, line := range lines {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch strings.ToLower(key) {
		case "description":
			meta.Description = strings.Trim(val, "\"")
		case "version":
			meta.Version = strings.Trim(val, "\"")
		case "author":
			meta.Author = strings.Trim(val, "\"")
		case "tags":
			meta.Tags = parseTags(val)
		}
	}

	return meta
}

func extractFrontmatter(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	if !scanner.Scan() || scanner.Text() != "---" {
		return ""
	}

	var lines []string
	for scanner.Scan() {
		if scanner.Text() == "---" {
			break
		}
		lines = append(lines, scanner.Text())
	}

	return strings.Join(lines, "\n")
}

func parseTags(val string) []string {
	val = strings.Trim(val, "[]")
	parts := strings.Split(val, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		t = strings.Trim(t, "\"'")
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
