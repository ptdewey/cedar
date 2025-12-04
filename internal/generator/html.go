package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ptdewey/cedar/internal/config"
	"github.com/ptdewey/cedar/internal/parser"
)

// Note: fields in this struct are used inside HTML templates, and must be exported.
type htmlPage struct {
	Metadata    map[string]any
	HTMLContent template.HTML
	AllPages    map[string][]PageInfo // Pages organized by route contentPath
}

// PageInfo is a simplified version of [parser.Page] for template access
type PageInfo struct {
	Metadata map[string]any
	Slug     string
	Date     time.Time // For sorting
}

// LinkItem represents a link with title for template partials
type LinkItem struct {
	Link  string
	Title string
}

// WritingItem represents a writing entry with all display fields
type WritingItem struct {
	Date  string
	Link  string
	Title string
	Type  string
}

func WriteHTMLFiles(pages []parser.Page, outputDir string, cfg *config.Config) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Organize pages by route for easy template access
	pagesByRoute := make(map[string][]PageInfo)
	for _, p := range pages {
		if p.Route != nil {
			slug, _ := p.Metadata["slug"].(string)
			info := PageInfo{
				Metadata: p.Metadata,
				Slug:     slug,
				Date:     p.Date,
			}
			pagesByRoute[p.Route.ContentPath] = append(pagesByRoute[p.Route.ContentPath], info)
		}
	}

	// Sort pages within each route by date (descending)
	for route := range pagesByRoute {
		sort.Slice(pagesByRoute[route], func(i, j int) bool {
			return pagesByRoute[route][i].Date.After(pagesByRoute[route][j].Date)
		})
	}

	for _, page := range pages {
		if err := writePage(page, pagesByRoute, outputDir, cfg); err != nil {
			return err
		}
	}

	return nil
}

func writePage(page parser.Page, pagesByRoute map[string][]PageInfo, outputDir string, cfg *config.Config) error {
	if page.Route == nil {
		return nil
	}

	templatePath := filepath.Join(cfg.TemplateDir, page.Route.Template)

	// Create an empty template set with helper functions
	t := template.New("").Funcs(template.FuncMap{
		"linkItem": func(link, title string) LinkItem {
			return LinkItem{Link: link, Title: title}
		},
		"getStr": func(m map[string]any, key string) string {
			if v, ok := m[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		},
		"writingItems": func(pages []PageInfo) []WritingItem {
			items := make([]WritingItem, 0, len(pages))
			for _, p := range pages {
				date := ""
				if d, ok := p.Metadata["date"].(string); ok {
					date = d
				}
				title := ""
				if t, ok := p.Metadata["title"].(string); ok {
					title = t
				}
				typ := ""
				if t, ok := p.Metadata["type"].(string); ok {
					typ = t
				}

				items = append(items, WritingItem{
					Date:  date,
					Link:  fmt.Sprintf("/writing/%s", p.Slug),
					Title: title,
					Type:  typ,
				})
			}
			return items
		},
		"formatDate": func(dateStr string) string {
			t, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return dateStr
			}
			return t.Format("Jan 2, 2006")
		},
	})

	// Parse base template first
	var err error
	if cfg.BaseTemplatePath != "" {
		basePath := filepath.Join(cfg.TemplateDir, cfg.BaseTemplatePath)
		t, err = t.ParseFiles(basePath)
		if err != nil {
			return err
		}
	}

	// Parse all partials
	partialsPattern := filepath.Join(cfg.TemplateDir, "partials", "*"+cfg.TemplateExt)
	t, err = t.ParseGlob(partialsPattern)
	if err != nil {
		return err
	}

	// Parse the page template last
	t, err = t.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	data := htmlPage{
		Metadata:    page.Metadata,
		HTMLContent: template.HTML(page.Content),
		AllPages:    pagesByRoute,
	}

	var buf bytes.Buffer
	// Execute the specific page template by name
	templateName := filepath.Base(templatePath)
	if err := t.ExecuteTemplate(&buf, templateName, data); err != nil {
		return err
	}

	outputPath := parser.GetOutputPath(page, outputDir)
	outputPathDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputPathDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
