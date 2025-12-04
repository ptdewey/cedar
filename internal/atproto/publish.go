package atproto

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ptdewey/cedar/internal/config"
	"github.com/ptdewey/cedar/internal/parser"
	libleaflet "github.com/ptdewey/standard-site-go/leaflet"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// newMDParser returns a goldmark parser configured with GFM and Footnote extensions.
func newMDParser() gparser.Parser {
	return goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote)).Parser()
}

// DryRun prints the records that would be published as JSON without making
// any API calls. Useful for inspecting the output before authenticating.
func DryRun(cfg *config.Config, pages []parser.Page) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	mdParser := newMDParser()

	for pubKey, pub := range cfg.ATProto.Publications {
		pubRecord := buildPublicationRecord(pub)
		fmt.Printf("=== site.standard.publication: %s ===\n", pubKey)
		if err := enc.Encode(pubRecord); err != nil {
			return err
		}

		pubURI := fmt.Sprintf("at://dry-run/site.standard.publication/%s", pubKey)

		for _, page := range pages {
			if page.Route == nil || page.Route.Publish != pubKey {
				continue
			}

			source := []byte(page.RawMarkdown)
			astDoc := mdParser.Parse(text.NewReader(source))
			leafletDoc := libleaflet.Convert(source, astDoc)

			title, _ := page.Metadata["title"].(string)
			fmt.Printf("\n=== site.standard.document: %s ===\n", title)
			if err := enc.Encode(buildDocumentRecord(page, pubURI, cfg, leafletDoc, "")); err != nil {
				return err
			}
		}
	}

	return nil
}

// Preview generates HTML preview files for leaflet block rendering.
func Preview(cfg *config.Config, pages []parser.Page, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating preview directory: %w", err)
	}

	mdParser := newMDParser()

	var count int
	for _, page := range pages {
		if page.Route == nil || page.Route.Publish == "" {
			continue
		}

		source := []byte(page.RawMarkdown)
		doc := mdParser.Parse(text.NewReader(source))
		content := libleaflet.Convert(source, doc)
		body := libleaflet.RenderHTML(content)
		htmlOut := "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"></head><body>\n" + body + "\n</body></html>\n"

		slug := strings.TrimSuffix(filepath.Base(page.SourcePath), filepath.Ext(page.SourcePath))
		outPath := filepath.Join(outDir, slug+".html")
		if err := os.WriteFile(outPath, []byte(htmlOut), 0o644); err != nil {
			return fmt.Errorf("writing preview for %s: %w", slug, err)
		}

		title, _ := page.Metadata["title"].(string)
		fmt.Printf("  Preview: %s -> %s\n", title, outPath)
		count++
	}

	fmt.Printf("\n%d preview file(s) written to %s/\n", count, outDir)
	return nil
}

// Publish syncs content to the ATProto PDS as site.standard records.
func Publish(cfg *config.Config, pages []parser.Page) error {
	ctx := context.Background()

	sess, err := LoadSession()
	if err != nil {
		return fmt.Errorf("loading auth state (run 'cedar auth' first): %w", err)
	}

	state, err := LoadPublishState()
	if err != nil {
		return fmt.Errorf("loading publish state: %w", err)
	}

	store := newFileAuthStore()
	client, err := NewClient(sess, store)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	// Ensure all configured publications exist
	for pubKey, pub := range cfg.ATProto.Publications {
		if _, exists := state.Publications[pubKey]; exists {
			continue
		}

		var uri, rkey string
		if pub.RKey != "" {
			// Pin to an existing record — just record the AT-URI without touching the record.
			rkey = pub.RKey
			uri = fmt.Sprintf("at://%s/site.standard.publication/%s", sess.AccountDID, rkey)
			fmt.Printf("Using pinned publication %q (%s)\n", pubKey, uri)
		} else {
			fmt.Printf("Creating publication %q...\n", pubKey)
			record := buildPublicationRecord(pub)
			var err error
			uri, rkey, err = client.CreateRecord(ctx, "site.standard.publication", record)
			if err != nil {
				return fmt.Errorf("creating publication %q: %w", pubKey, err)
			}
			fmt.Printf("  Created: %s\n  Add 'rkey = %q' to your config to reuse this record.\n", uri, rkey)
		}
		state.Publications[pubKey] = PublicationState{ATURI: uri, RKey: rkey}
		if err := state.Save(); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}
	}

	mdParser := newMDParser()

	// Sync documents per publication
	var created, updated, skipped int
	for pubKey := range cfg.ATProto.Publications {
		pubState := state.Publications[pubKey]

		for _, page := range pages {
			if page.Route == nil || page.Route.Publish != pubKey {
				continue
			}

			relPath, err := filepath.Rel(cfg.ContentDir, page.SourcePath)
			if err != nil {
				relPath = page.SourcePath
			}

			title, _ := page.Metadata["title"].(string)
			contentHash := fmt.Sprintf("%x", md5.Sum([]byte(page.RawMarkdown)))

			existing, exists := state.Documents[relPath]
			if exists && existing.ContentHash == contentHash && existing.Publication == pubKey {
				skipped++
				continue
			}

			source := []byte(page.RawMarkdown)
			astDoc := mdParser.Parse(text.NewReader(source))
			leafletDoc := libleaflet.Convert(source, astDoc)

			if exists && existing.Publication == pubKey {
				fmt.Printf("  Updating: %s\n", title)
				record := buildDocumentRecord(page, pubState.ATURI, cfg, leafletDoc, "/"+existing.RKey)
				if err := client.PutRecord(ctx, "site.standard.document", existing.RKey, record); err != nil {
					return fmt.Errorf("updating document %q: %w", title, err)
				}
				state.Documents[relPath] = DocumentRecord{
					Publication: pubKey,
					ATURI:       existing.ATURI,
					RKey:        existing.RKey,
					ContentHash: contentHash,
				}
				updated++
			} else {
				fmt.Printf("  Creating: %s\n", title)
				// Create without path first to get the rkey assigned by the PDS.
				record := buildDocumentRecord(page, pubState.ATURI, cfg, leafletDoc, "")
				uri, rkey, err := client.CreateRecord(ctx, "site.standard.document", record)
				if err != nil {
					return fmt.Errorf("creating document %q: %w", title, err)
				}
				// Update the record immediately with the correct path now that rkey is known.
				record.Path = "/" + rkey
				if err := client.PutRecord(ctx, "site.standard.document", rkey, record); err != nil {
					return fmt.Errorf("setting path for document %q: %w", title, err)
				}
				state.Documents[relPath] = DocumentRecord{
					Publication: pubKey,
					ATURI:       uri,
					RKey:        rkey,
					ContentHash: contentHash,
				}
				created++
			}

			if err := state.Save(); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
		}
	}

	fmt.Printf("\nPublish complete: %d created, %d updated, %d unchanged\n", created, updated, skipped)
	if created > 0 || updated > 0 {
		fmt.Println("Run 'cedar build' to update verification endpoints.")
	}
	return nil
}
