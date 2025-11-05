package parser

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/pdewey/cedar/internal/highlighter"
	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type Page struct {
	Metadata map[string]any `json:"metadata"`
	Content  string         `json:"content"`
}

func generateSlug(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, ".", "")
	return slug
}

func parseFrontMatter(content []byte) (map[string]any, []byte, error) {
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "---") {
		return nil, content, nil
	}
	parts := strings.SplitN(contentStr, "---", 3)
	if len(parts) < 3 {
		return nil, content, fmt.Errorf("invalid front-matter format")
	}
	var metadata map[string]any
	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		return nil, nil, err
	}
	return metadata, []byte(parts[2]), nil
}

func getReadingTime(text string) int {
	words := strings.Fields(text)
	wordCount := len(words)

	// reading/speaking rate
	wordsPerMinute := 200.0
	return int(math.Round(float64(wordCount) / wordsPerMinute))
}
func ProcessMarkdownFile(path string) (Page, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}

	metadata, markdownContent, err := parseFrontMatter(content)
	if err != nil {
		return Page{}, err
	} else if metadata == nil {
		_, filename := filepath.Split(path)
		filename = strings.SplitN(filename, ".", 1)[0]
		// TODO: allow configuration of default metadata
		metadata = map[string]any{
			"title":       filename,
			"description": "",
			"date":        time.Now().Format("2006-01-02"),
		}
	}

	if _, ok := metadata["slug"]; !ok {
		fname := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		metadata["slug"] = generateSlug(fname)
	}

	var htmlContent bytes.Buffer

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			highlighter.NewTreeSitterExtension(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	if err := md.Convert(markdownContent, &htmlContent); err != nil {
		return Page{}, err
	}

	metadata["read_time"] = getReadingTime(string(markdownContent))

	return Page{
		Metadata: metadata,
		Content:  htmlContent.String(),
	}, nil
}

func ProcessDirectory(dir string) ([]Page, error) {
	var writings []Page

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(d.Name(), ".md") {
			page, err := ProcessMarkdownFile(path)
			if err != nil {
				return err
			}

			writings = append(writings, page)
		}
		return nil
	})

	return writings, err
}
