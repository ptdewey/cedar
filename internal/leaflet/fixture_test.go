package leaflet

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var update = flag.Bool("update", false, "fetch fresh Leaflet fixtures from the ATProto network")

// fixtureTargets lists public ATProto accounts that publish Leaflet content.
// Records from pub.leaflet.content are fetched when -update is passed.
var fixtureTargets = []struct {
	handle     string
	collection string
	limit      int
}{
	{"pfrazee.com", "pub.leaflet.content", 5},
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		ctx := context.Background()
		if err := updateFixtures(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "fixture update: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// TestFixtures loads all JSON files from testdata/leaflet/ and validates that
// every facet satisfies the byte-range invariants expected by the Leaflet spec.
func TestFixtures(t *testing.T) {
	dir := filepath.Join("testdata", "leaflet")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		t.Skip("testdata/leaflet/ not found — run with -update to fetch fixtures")
	}
	if err != nil {
		t.Fatal(err)
	}

	var ran int
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		ran++
		t.Run(entry.Name(), func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			var raw map[string]any
			if err := json.Unmarshal(b, &raw); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			validateRawContent(t, raw)
		})
	}

	if ran == 0 {
		t.Skip("no JSON files in testdata/leaflet/ — run with -update to fetch fixtures")
	}
}

// TestConverterInvariants runs the invariant checks against the output of our
// own converter across a broad range of markdown inputs.  This is always
// exercised (no fixtures required) and catches regressions in the converter.
func TestConverterInvariants(t *testing.T) {
	cases := []string{
		// Basic inline formatting
		"**bold**",
		"*italic*",
		"***both***",
		"~~strike~~",
		"`code`",
		// Links
		"[link](https://example.com)",
		"[**bold** link](https://example.com)",
		"[*italic* and **bold**](https://example.com)",
		// Headings
		"# H1",
		"## H2 with **bold**",
		"###### H6",
		// Code blocks
		"```go\nfmt.Println(\"hello\")\n```",
		"```\nno language\n```",
		"    indented code\n    second line",
		// Blockquotes
		"> simple quote",
		"> **bold** in quote",
		"> first\n>\n> **second** paragraph",
		// Lists
		"- item one\n- item two\n- item three",
		"- parent\n  - child1\n  - child2",
		"- level1\n  - level2\n    - level3",
		"1. first\n2. second\n3. third",
		"3. third\n4. fourth",
		"- unordered\n  1. nested ordered\n  2. also ordered",
		"1. ordered\n   - nested unordered\n   - also unordered",
		// Hard line breaks (trailing double-space)
		"line one  \nline two",
		// Misc
		"---",
		"",
		"plain text",
		// Multi-byte Unicode
		"**café**",
		"**日本語**",
		"café **bold**",
		"**🎉**",
		"🎉 **party** time",
		// Mixed documents
		"# Title\n\nSome **bold** text.\n\n- item\n\n```go\ncode\n```\n\n---\n\n> quote",
	}

	for i, md := range cases {
		name := fmt.Sprintf("case_%02d", i)
		md := md
		t.Run(name, func(t *testing.T) {
			content, _ := parse(md)
			for _, page := range content.Pages {
				for _, entry := range page.Blocks {
					checkBlockInvariants(t, entry.Block)
				}
			}
			// Round-trip through JSON: marshal → unmarshal → validate raw
			b, err := json.Marshal(content)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var raw map[string]any
			if err := json.Unmarshal(b, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			validateRawContent(t, raw)
		})
	}
}

// ---------------------------------------------------------------------------
// Raw JSON validators
//
// These operate on map[string]any so they work on both hand-crafted fixtures
// and network-fetched records without needing a typed deserialiser.
// ---------------------------------------------------------------------------

func validateRawContent(t *testing.T, raw map[string]any) {
	t.Helper()
	pages, _ := raw["pages"].([]any)
	for i, page := range pages {
		pageMap, _ := page.(map[string]any)
		if pageMap == nil {
			continue
		}
		blocks, _ := pageMap["blocks"].([]any)
		for j, entry := range blocks {
			entryMap, _ := entry.(map[string]any)
			if entryMap == nil {
				continue
			}
			block, _ := entryMap["block"].(map[string]any)
			t.Run(fmt.Sprintf("page%d_block%d", i, j), func(t *testing.T) {
				validateRawBlock(t, block)
			})
		}
	}
}

func validateRawBlock(t *testing.T, block map[string]any) {
	t.Helper()
	if block == nil {
		return
	}
	blockType, _ := block["$type"].(string)
	switch blockType {
	case "pub.leaflet.blocks.text",
		"pub.leaflet.blocks.header",
		"pub.leaflet.blocks.blockquote":
		plaintext, _ := block["plaintext"].(string)
		facets, _ := block["facets"].([]any)
		checkRawFacetInvariants(t, plaintext, facets)

	case "pub.leaflet.blocks.unorderedList",
		"pub.leaflet.blocks.orderedList":
		children, _ := block["children"].([]any)
		for _, child := range children {
			if childMap, ok := child.(map[string]any); ok {
				validateRawListItem(t, childMap)
			}
		}
	}
}

func validateRawListItem(t *testing.T, item map[string]any) {
	t.Helper()
	if content, ok := item["content"].(map[string]any); ok {
		validateRawBlock(t, content)
	}
	if children, ok := item["children"].([]any); ok {
		for _, child := range children {
			if m, ok := child.(map[string]any); ok {
				validateRawListItem(t, m)
			}
		}
	}
	if nested, ok := item["orderedListChildren"].(map[string]any); ok {
		validateRawBlock(t, nested)
	}
	if nested, ok := item["unorderedListChildren"].(map[string]any); ok {
		validateRawBlock(t, nested)
	}
}

// checkRawFacetInvariants checks the same invariants as checkFacetInvariants
// but operates on raw JSON values (float64 indices, any features).
func checkRawFacetInvariants(t *testing.T, plaintext string, facets []any) {
	t.Helper()
	pb := []byte(plaintext)
	n := len(pb)
	for i, facet := range facets {
		facetMap, ok := facet.(map[string]any)
		if !ok {
			t.Errorf("facet[%d]: not an object", i)
			continue
		}
		index, ok := facetMap["index"].(map[string]any)
		if !ok {
			t.Errorf("facet[%d]: missing index object", i)
			continue
		}
		startF, hasStart := index["byteStart"].(float64)
		endF, hasEnd := index["byteEnd"].(float64)
		if !hasStart || !hasEnd {
			t.Errorf("facet[%d]: missing byteStart/byteEnd", i)
			continue
		}
		s, e := int(startF), int(endF)
		if s < 0 {
			t.Errorf("facet[%d]: negative byteStart %d", i, s)
		}
		if e < s {
			t.Errorf("facet[%d]: byteEnd %d < byteStart %d", i, e, s)
		}
		if e > n {
			t.Errorf("facet[%d]: byteEnd %d > plaintext len %d", i, e, n)
		}
		if s > 0 && s < n && !utf8.RuneStart(pb[s]) {
			t.Errorf("facet[%d]: byteStart %d splits a UTF-8 codepoint (0x%02x)", i, s, pb[s])
		}
		if e > 0 && e < n && !utf8.RuneStart(pb[e]) {
			t.Errorf("facet[%d]: byteEnd %d splits a UTF-8 codepoint (0x%02x)", i, e, pb[e])
		}
	}
}

// ---------------------------------------------------------------------------
// Network fixture fetchers (used only when -update is passed)
// ---------------------------------------------------------------------------

func updateFixtures(ctx context.Context) error {
	dir := filepath.Join("testdata", "leaflet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating fixture dir: %w", err)
	}

	for _, target := range fixtureTargets {
		fmt.Printf("fetching %s/%s ...\n", target.handle, target.collection)

		did, err := resolveHandle(ctx, target.handle)
		if err != nil {
			return fmt.Errorf("resolve handle %s: %w", target.handle, err)
		}

		pds, err := resolvePDS(ctx, did)
		if err != nil {
			return fmt.Errorf("resolve PDS for %s: %w", did, err)
		}

		records, err := listRecords(ctx, pds, did, target.collection, target.limit)
		if err != nil {
			return fmt.Errorf("list records for %s: %w", target.handle, err)
		}

		safe := strings.NewReplacer(".", "_", ":", "_").Replace(target.handle)
		for i, rec := range records {
			b, err := json.MarshalIndent(rec, "", "  ")
			if err != nil {
				return err
			}
			path := filepath.Join(dir, fmt.Sprintf("%s_%02d.json", safe, i))
			if err := os.WriteFile(path, b, 0o644); err != nil {
				return err
			}
			fmt.Printf("  wrote %s\n", path)
		}
	}
	return nil
}

// resolveHandle resolves an ATProto handle to a DID.
// It first tries the .well-known/atproto-did endpoint and falls back to the
// com.atproto.identity.resolveHandle XRPC method on bsky.social for handles
// whose hosts redirect or don't serve the endpoint directly.
func resolveHandle(ctx context.Context, handle string) (string, error) {
	u := fmt.Sprintf("https://%s/.well-known/atproto-did", handle)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err == nil {
		if resp, err := http.DefaultClient.Do(req); err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if did := strings.TrimSpace(string(b)); strings.HasPrefix(did, "did:") {
					return did, nil
				}
			}
		}
	}

	// Fall back to XRPC resolution via the public AppView.
	params := url.Values{}
	params.Set("handle", handle)
	xrpcURL := "https://bsky.social/xrpc/com.atproto.identity.resolveHandle?" + params.Encode()
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, xrpcURL, nil)
	if err != nil {
		return "", err
	}
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", fmt.Errorf("XRPC resolveHandle %s: %w", handle, err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp2.Body)
		return "", fmt.Errorf("XRPC resolveHandle %s: HTTP %d: %s", handle, resp2.StatusCode, string(b))
	}
	var result struct {
		DID string `json:"did"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.DID == "" {
		return "", fmt.Errorf("empty DID for handle %s", handle)
	}
	return result.DID, nil
}

func resolvePDS(ctx context.Context, did string) (string, error) {
	var docURL string
	if strings.HasPrefix(did, "did:plc:") {
		docURL = "https://plc.directory/" + did
	} else if strings.HasPrefix(did, "did:web:") {
		domain := strings.TrimPrefix(did, "did:web:")
		docURL = "https://" + domain + "/.well-known/did.json"
	} else {
		return "", fmt.Errorf("unsupported DID method in %s", did)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, docURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching DID document for %s", resp.StatusCode, did)
	}

	var doc struct {
		Service []struct {
			ID              string `json:"id"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	for _, svc := range doc.Service {
		if svc.ID == "#atproto_pds" {
			return strings.TrimRight(svc.ServiceEndpoint, "/"), nil
		}
	}
	return "", fmt.Errorf("no PDS service in DID document for %s", did)
}

// listRecords fetches up to limit records from a collection using an
// unauthenticated com.atproto.repo.listRecords call.
func listRecords(ctx context.Context, pds, did, collection string, limit int) ([]map[string]any, error) {
	params := url.Values{}
	params.Set("repo", did)
	params.Set("collection", collection)
	params.Set("limit", fmt.Sprintf("%d", limit))
	endpoint := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords?%s", pds, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Records []struct {
			Value map[string]any `json:"value"`
		} `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(result.Records))
	for _, r := range result.Records {
		out = append(out, r.Value)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Real markdown file conversion test
// ---------------------------------------------------------------------------

// parseFullPipeline parses markdown with GFM + Footnote extensions, matching
// the same goldmark configuration used by the publish/preview pipeline.
func parseFullPipeline(md string) *Document {
	source := []byte(md)
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
	).Parser()
	doc := parser.Parse(text.NewReader(source))
	return Convert(source, doc)
}

// stripFrontmatter removes YAML front matter (---...---) from markdown.
func stripFrontmatter(src string) string {
	if !strings.HasPrefix(src, "---") {
		return src
	}
	parts := strings.SplitN(src, "---", 3)
	if len(parts) < 3 {
		return src
	}
	return strings.TrimSpace(parts[2])
}

// TestConvertRealMarkdown converts the project's bluesky-comments-svelte.md
// through the full pipeline and validates that:
//   - all facet byte-range invariants hold
//   - expected block types and content are present
//   - RenderHTML produces non-empty output without panicking
func TestConvertRealMarkdown(t *testing.T) {
	raw, err := os.ReadFile("./testdata/bluesky-comments-svelte.md")
	if err != nil {
		t.Fatalf("reading source file: %v", err)
	}

	md := stripFrontmatter(string(raw))
	content := parseFullPipeline(md)

	if len(content.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(content.Pages))
	}
	blocks := content.Pages[0].Blocks

	// Log block summary (visible with -v)
	for i, entry := range blocks {
		switch b := entry.Block.(type) {
		case HeaderBlock:
			t.Logf("[%2d] HEADER h%d  %q", i, b.Level, b.Plaintext)
		case TextBlock:
			preview := b.Plaintext
			if len(preview) > 60 {
				preview = preview[:60] + "…"
			}
			t.Logf("[%2d] TEXT        %q  (%d facets)", i, preview, len(b.Facets))
		case CodeBlock:
			preview := b.Plaintext
			if len(preview) > 40 {
				preview = preview[:40]
			}
			t.Logf("[%2d] CODE        lang=%q  %q", i, b.Language, preview)
		case HorizontalRuleBlock:
			t.Logf("[%2d] HR", i)
		case UnorderedListBlock:
			t.Logf("[%2d] UL          (%d items)", i, len(b.Children))
		case OrderedListBlock:
			t.Logf("[%2d] OL          (%d items)", i, len(b.Children))
		default:
			t.Logf("[%2d] UNKNOWN     %T", i, b)
		}
	}

	// Invariant check on every block
	for _, entry := range blocks {
		checkBlockInvariants(t, entry.Block)
	}

	// Structural checks: expected block types in order
	t.Run("has_headers", func(t *testing.T) {
		var headers []HeaderBlock
		for _, e := range blocks {
			if h, ok := e.Block.(HeaderBlock); ok {
				headers = append(headers, h)
			}
		}
		if len(headers) == 0 {
			t.Fatal("no header blocks found")
		}
		// First header should be "Introduction" (### = level 3)
		if headers[0].Plaintext != "Introduction" {
			t.Errorf("first header = %q, want %q", headers[0].Plaintext, "Introduction")
		}
		if headers[0].Level != 3 {
			t.Errorf("first header level = %d, want 3", headers[0].Level)
		}
	})

	t.Run("has_code_blocks", func(t *testing.T) {
		var codeBlocks []CodeBlock
		for _, e := range blocks {
			if cb, ok := e.Block.(CodeBlock); ok {
				codeBlocks = append(codeBlocks, cb)
			}
		}
		if len(codeBlocks) == 0 {
			t.Fatal("no code blocks found")
		}
		// First code block is the `npm install` shell command
		if codeBlocks[0].Language != "sh" {
			t.Errorf("first code block language = %q, want %q", codeBlocks[0].Language, "sh")
		}
		if !strings.Contains(codeBlocks[0].Plaintext, "bluesky-comments-svelte") {
			t.Errorf("first code block plaintext = %q, want npm install command", codeBlocks[0].Plaintext)
		}
	})

	t.Run("has_link_facets", func(t *testing.T) {
		var linkCount int
		for _, e := range blocks {
			if tb, ok := e.Block.(TextBlock); ok {
				for _, f := range tb.Facets {
					if f.Features[0].Type == "pub.leaflet.richtext.facet#link" {
						linkCount++
					}
				}
			}
		}
		if linkCount == 0 {
			t.Error("no link facets found in any text block")
		}
	})

	t.Run("render_html_nonempty", func(t *testing.T) {
		html := RenderHTML(content)
		if len(html) == 0 {
			t.Error("RenderHTML returned empty string")
		}
		if !strings.Contains(html, "Introduction") {
			t.Error("rendered HTML missing expected heading text")
		}
		if !strings.Contains(html, "bluesky-comments-svelte") {
			t.Error("rendered HTML missing expected code content")
		}
	})

	t.Run("json_round_trip", func(t *testing.T) {
		b, err := json.Marshal(content)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var raw map[string]any
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		validateRawContent(t, raw)
	})
}
