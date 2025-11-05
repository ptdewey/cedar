package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"

	"codeberg.org/pdewey/cedar/internal/parser"
	"codeberg.org/pdewey/cedar/internal/rss"
)

func writeJSONFile(data any, outputPath string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, jsonData, 0644)
}

// TODO: refactor to split html converter and writer
func writeHTMLFiles(pages []parser.Page, outputDir string, templatePath string) error {
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	for _, page := range pages {
		slug, ok := page.Metadata["slug"].(string)
		if !ok || slug == "" {
			continue
		}

		data := struct {
			Metadata    map[string]any
			HTMLContent template.HTML
		}{
			Metadata:    page.Metadata,
			HTMLContent: template.HTML(page.Content),
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return err
		}

		outputPath := filepath.Join(outputDir, slug+".html")
		if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	writings, err := parser.ProcessDirectory("content")
	if err != nil {
		fmt.Printf("Error processing directory: %v\n", err)
		os.Exit(1)
	}

	sort.Slice(writings, func(i, j int) bool {
		dateI, _ := time.Parse("2006-01-02", writings[i].Metadata["date"].(string))
		dateJ, _ := time.Parse("2006-01-02", writings[j].Metadata["date"].(string))
		return dateI.After(dateJ)
	})

	if err := writeHTMLFiles(writings, "static/writing", "templates/page.html"); err != nil {
		fmt.Printf("Error writing HTML files: %v\n", err)
		os.Exit(1)
	}

	if err := rss.GenerateRSS(writings, "static/rss.xml"); err != nil {
		fmt.Printf("Error writing rss.xml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated HTML files and rss.xml")
}
