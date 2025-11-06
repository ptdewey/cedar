package rss

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"codeberg.org/pdewey/cedar/internal/parser"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category,omitempty"`
	Content     string `xml:"content"`
}

func GenerateRSS(pages []parser.Page, outputPath string) error {
	// TODO: rip this into some sort of config (probably toml)
	channel := Channel{
		Title:       "A website",
		Link:        "https://example.com",
		Description: "RSS feed for a website",
		PubDate:     time.Now().Format(time.RFC1123Z),
	}

	for _, page := range pages {
		var categories []string
		if rawCategories, ok := page.Metadata["categories"].([]any); ok {
			for _, category := range rawCategories {
				if strCategory, ok := category.(string); ok {
					categories = append(categories, strCategory)
				}
			}
		}

		description, ok := page.Metadata["description"].(string)
		if !ok {
			description = ""
		}

		item := Item{
			Title:       page.Metadata["title"].(string),
			Link:        fmt.Sprintf("https://example.com/writing/%s", page.Metadata["slug"].(string)),
			Description: description,
			Content:     page.Content,
			PubDate:     page.Metadata["date"].(string),
			Category:    strings.Join(categories, ", "),
		}
		channel.Items = append(channel.Items, item)
	}

	rss := RSS{
		Version: "2.0",
		Channel: channel,
	}

	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return err
	}

	rssHeader := []byte(xml.Header)
	output = append(rssHeader, output...)

	return os.WriteFile(outputPath, output, 0644)
}
