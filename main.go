package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"codeberg.org/pdewey/cedar/internal/generator"
	"codeberg.org/pdewey/cedar/internal/parser"
	"codeberg.org/pdewey/cedar/internal/rss"
)

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

	if err := generator.WriteHTMLFiles(writings, "static/writing", "templates/page.html"); err != nil {
		fmt.Printf("Error writing HTML files: %v\n", err)
		os.Exit(1)
	}

	if err := rss.GenerateRSS(writings, "static/rss.xml"); err != nil {
		fmt.Printf("Error writing rss.xml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated HTML files and rss.xml")
}
