package main

import (
	"flag"
	"fmt"
	"os"

	"codeberg.org/pdewey/cedar/internal/config"
	"codeberg.org/pdewey/cedar/internal/generator"
	"codeberg.org/pdewey/cedar/internal/parser"
	"codeberg.org/pdewey/cedar/internal/rss"
)

var (
	flagConfigPath = flag.String("config", "cedar.toml", "-config 'cedar.toml'")
)

func main() {
	flag.Parse()

	cfg, err := config.Parse(*flagConfigPath)
	if err != nil {
		fmt.Printf("failed to parse configuration: %v\n", err)
		os.Exit(1)
	}

	pages, err := parser.ProcessDirectory(cfg)
	if err != nil {
		fmt.Printf("failed to process content directory: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML files, writing them to the build cache directory.
	if err := generator.WriteHTMLFiles(pages, cfg.CacheDir, cfg.TemplateDir); err != nil {
		fmt.Printf("Error writing HTML files: %v\n", err)
		os.Exit(1)
	}

	// Clean publish directory if clean build is enabled.
	if cfg.CleanBuild {
		if err := os.RemoveAll(cfg.PublishDir); err != nil {
			fmt.Printf("failed to removed publish directory '%s': %v\n", cfg.PublishDir, err)
		}
	}

	// Copy static HTML files from build cache to publish directory.
	if err := generator.CopyDirIncremental(cfg.CacheDir, cfg.PublishDir); err != nil {
		fmt.Printf("error copying static directory: %v\n", err)
		os.Exit(1)
	} else {
		_ = os.RemoveAll(cfg.CacheDir)
	}

	// Copy other static files.
	if err := generator.CopyDirIncremental(cfg.StaticDir, cfg.PublishDir); err != nil {
		fmt.Printf("error copying static directory: %v\n", err)
		os.Exit(1)
	}

	// Generate RSS feed if enabled.
	if cfg.RSS.Generate {
		if err := rss.GenerateRSS(pages, cfg); err != nil {
			fmt.Printf("Error writing rss.xml: %v\n", err)
			os.Exit(1)
		}
	}

	msg := "Successfully generated HTML files"
	if cfg.RSS.Generate {
		msg += " and rss.xml"
	}
	fmt.Println(msg)
}
