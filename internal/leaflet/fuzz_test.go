package leaflet

import (
	"testing"
	"unicode/utf8"
)

// FuzzConvert feeds arbitrary markdown through the converter and asserts
// invariants that must always hold on the output:
//   - no panic
//   - every facet's byteStart <= byteEnd <= len(UTF-8 plaintext)
//   - both byteStart and byteEnd land on valid UTF-8 codepoint boundaries
//   - RenderHTML does not panic
func FuzzConvert(f *testing.F) {
	// Seed corpus with the cases already covered by the unit tests so the
	// fuzzer starts from a useful baseline.
	seeds := []string{
		"**bold**",
		"*italic*",
		"***both***",
		"~~strike~~",
		"`code`",
		"[link](https://example.com)",
		"[**bold** in link](https://example.com)",
		"# H1",
		"## Heading with **bold**",
		"```go\nfmt.Println(\"hello\")\n```",
		"    indented code\n    second line",
		"- item one\n- item two\n  - nested",
		"1. first\n2. second",
		"- item\n  1. nested ordered",
		"> blockquote",
		"> **bold** in quote\n>\n> second paragraph",
		"---",
		"**café**",
		"**日本語**",
		"café **bold**",
		"**🎉**",
		"line one  \nline two",
		"# Title\n\nParagraph **bold**.\n\n- item\n\n```go\ncode\n```\n\n---",
		"",
		"plain text",
		"<div>html</div>\n\nregular",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, md string) {
		content, _ := parse(md)
		if content == nil {
			return
		}
		for _, page := range content.Pages {
			for _, entry := range page.Blocks {
				checkBlockInvariants(t, entry.Block)
			}
		}
		// Must not panic
		_ = RenderHTML(content)
	})
}

// checkBlockInvariants recursively validates a block and all its descendants.
func checkBlockInvariants(t *testing.T, block any) {
	t.Helper()
	switch b := block.(type) {
	case TextBlock:
		checkFacetInvariants(t, b.Plaintext, b.Facets)
	case HeaderBlock:
		checkFacetInvariants(t, b.Plaintext, b.Facets)
	case BlockquoteBlock:
		checkFacetInvariants(t, b.Plaintext, b.Facets)
	case UnorderedListBlock:
		for i := range b.Children {
			checkListItemInvariants(t, b.Children[i])
		}
	case OrderedListBlock:
		for i := range b.Children {
			checkListItemInvariants(t, b.Children[i])
		}
	}
}

func checkListItemInvariants(t *testing.T, item ListItem) {
	t.Helper()
	if item.Content != nil {
		checkBlockInvariants(t, item.Content)
	}
	for i := range item.Children {
		checkListItemInvariants(t, item.Children[i])
	}
	if item.OrderedListChildren != nil {
		for i := range item.OrderedListChildren.Children {
			checkListItemInvariants(t, item.OrderedListChildren.Children[i])
		}
	}
	if item.UnorderedListChildren != nil {
		for i := range item.UnorderedListChildren.Children {
			checkListItemInvariants(t, item.UnorderedListChildren.Children[i])
		}
	}
}

// checkFacetInvariants validates byte-range invariants on a set of facets.
func checkFacetInvariants(t *testing.T, plaintext string, facets []Facet) {
	t.Helper()
	pb := []byte(plaintext)
	n := len(pb)
	for i, f := range facets {
		s, e := f.Index.ByteStart, f.Index.ByteEnd
		if s < 0 {
			t.Errorf("facet[%d]: negative byteStart %d", i, s)
		}
		if e < s {
			t.Errorf("facet[%d]: byteEnd %d < byteStart %d", i, e, s)
		}
		if e > n {
			t.Errorf("facet[%d]: byteEnd %d > text len %d", i, e, n)
		}
		// byteStart must be at a valid UTF-8 codepoint boundary
		if s > 0 && s < n && !utf8.RuneStart(pb[s]) {
			t.Errorf("facet[%d]: byteStart %d splits a UTF-8 codepoint (byte=0x%02x)", i, s, pb[s])
		}
		// byteEnd must be at a valid UTF-8 codepoint boundary (or exactly at end)
		if e > 0 && e < n && !utf8.RuneStart(pb[e]) {
			t.Errorf("facet[%d]: byteEnd %d splits a UTF-8 codepoint (byte=0x%02x)", i, e, pb[e])
		}
	}
}
