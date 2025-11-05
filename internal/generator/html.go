package generator

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"

	"codeberg.org/pdewey/cedar/internal/parser"
)

// TODO: refactor to split html converter and writer
func WriteHTMLFiles(pages []parser.Page, outputDir string, templatePath string) error {
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
