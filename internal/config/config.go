package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/pelletier/go-toml/v2"
)

// TODO: need to support more complicated structure for dynamic routes
type Config struct {
	PublishDir       string
	StaticDir        string
	ContentDir       string
	TemplateDir      string
	TemplateExt      string
	CacheDir         string
	BaseTemplatePath string
	Copyright        string
	CleanBuild       bool
	BuildDraft       bool
	BuildFuture      bool
	AllowUnsafeHTML  bool
	RSS              RSS
	// TODO: atom feed support
	Routes []Route
}

type RSS struct {
	Generate    bool
	Title       string
	Description string
	URL         string
}

type Route struct {
	ContentPath   string
	OutputPattern string
	Template      string
	// TODO: atom feed support
	GenerateRSS bool
}

var defaultConfig = Config{
	PublishDir:       "public",
	StaticDir:        "static",
	ContentDir:       "content",
	TemplateDir:      "templates",
	TemplateExt:      ".tmpl",
	CacheDir:         "build",
	BaseTemplatePath: "",
	Copyright:        "",
	CleanBuild:       false,
	BuildDraft:       false,
	BuildFuture:      false,
	AllowUnsafeHTML:  false,
	RSS: RSS{
		Generate:    false,
		Title:       "Your Site",
		Description: "built with Cedar",
		URL:         "example.com",
	},
	Routes: []Route{
		{
			ContentPath:   "index.md",
			OutputPattern: "/",
			Template:      "_index.html",
		},
	},
}

type decoder interface {
	Decode(v any) error
}

func Parse(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)

	var d decoder
	switch ext := filepath.Ext(path); ext {
	case ".toml":
		d = toml.NewDecoder(buf)
	case ".json":
		d = json.NewDecoder(buf)
	case ".yaml":
		d = yaml.NewDecoder(buf)
	default:
		return nil, fmt.Errorf("invalid config file type: %s", ext)
	}

	cfg := defaultConfig
	if err := d.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
