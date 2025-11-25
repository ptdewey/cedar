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

	// TODO: clear public dir if clean is enabled
	// - should save new files to a cache and only remove after build is successful

	if err := generator.WriteHTMLFiles(pages, cfg.PublishDir, cfg.TemplateDir); err != nil {
		fmt.Printf("Error writing HTML files: %v\n", err)
		os.Exit(1)
	}

	// FIX: this doesn't work if any of the files already exist
	if err := os.CopyFS(cfg.PublishDir, os.DirFS(cfg.StaticDir)); err != nil {
		fmt.Printf("warn: error copying static directory: %v\n", err)
	}

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
