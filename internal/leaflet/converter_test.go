package leaflet

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func parse(md string) (*Document, []byte) {
	source := []byte(md)
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	).Parser()
	doc := parser.Parse(text.NewReader(source))
	return Convert(source, doc), source
}

func parseWithFootnotes(md string) (*Document, []byte) {
	source := []byte(md)
	parser := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
	).Parser()
	doc := parser.Parse(text.NewReader(source))
	return Convert(source, doc), source
}

func getBlock(t *testing.T, content *Document, index int) any {
	t.Helper()
	if len(content.Pages) == 0 {
		t.Fatal("no pages")
	}
	blocks := content.Pages[0].Blocks
	if index >= len(blocks) {
		t.Fatalf("block index %d out of range (have %d blocks)", index, len(blocks))
	}
	return blocks[index].Block
}

// --- Facet tests ---

func TestSimpleBold(t *testing.T) {
	content, _ := parse("**bold**")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "bold" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "bold")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
		t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
	}
	if f.Features[0].Type != "pub.leaflet.richtext.facet#bold" {
		t.Errorf("feature = %q, want bold", f.Features[0].Type)
	}
}

func TestSimpleItalic(t *testing.T) {
	content, _ := parse("*italic*")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "italic" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "italic")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 6 {
		t.Errorf("index = {%d,%d}, want {0,6}", f.Index.ByteStart, f.Index.ByteEnd)
	}
	if f.Features[0].Type != "pub.leaflet.richtext.facet#italic" {
		t.Errorf("feature = %q, want italic", f.Features[0].Type)
	}
}

func TestTripleEmphasis(t *testing.T) {
	content, _ := parse("***both***")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "both" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "both")
	}
	if len(b.Facets) != 2 {
		t.Fatalf("facets = %d, want 2", len(b.Facets))
	}
	// Both facets should cover {0,4}
	types := map[string]bool{}
	for _, f := range b.Facets {
		if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
			t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
		}
		types[f.Features[0].Type] = true
	}
	if !types["pub.leaflet.richtext.facet#bold"] {
		t.Error("missing bold facet")
	}
	if !types["pub.leaflet.richtext.facet#italic"] {
		t.Error("missing italic facet")
	}
}

func TestNestedBoldItalic(t *testing.T) {
	content, _ := parse("**bold *and italic***")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "bold and italic" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "bold and italic")
	}

	// Should have bold over all and italic over "and italic"
	var boldFacet, italicFacet *Facet
	for i := range b.Facets {
		switch b.Facets[i].Features[0].Type {
		case "pub.leaflet.richtext.facet#bold":
			boldFacet = &b.Facets[i]
		case "pub.leaflet.richtext.facet#italic":
			italicFacet = &b.Facets[i]
		}
	}
	if boldFacet == nil {
		t.Fatal("missing bold facet")
	}
	if italicFacet == nil {
		t.Fatal("missing italic facet")
	}
	if boldFacet.Index.ByteStart != 0 || boldFacet.Index.ByteEnd != 15 {
		t.Errorf("bold index = {%d,%d}, want {0,15}", boldFacet.Index.ByteStart, boldFacet.Index.ByteEnd)
	}
	if italicFacet.Index.ByteStart != 5 || italicFacet.Index.ByteEnd != 15 {
		t.Errorf("italic index = {%d,%d}, want {5,15}", italicFacet.Index.ByteStart, italicFacet.Index.ByteEnd)
	}
}

func TestLink(t *testing.T) {
	content, _ := parse("[click here](https://example.com)")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "click here" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "click here")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 10 {
		t.Errorf("index = {%d,%d}, want {0,10}", f.Index.ByteStart, f.Index.ByteEnd)
	}
	if f.Features[0].Type != "pub.leaflet.richtext.facet#link" {
		t.Errorf("feature = %q, want link", f.Features[0].Type)
	}
	if f.Features[0].URI != "https://example.com" {
		t.Errorf("uri = %q, want %q", f.Features[0].URI, "https://example.com")
	}
}

func TestBoldInsideLink(t *testing.T) {
	content, _ := parse("[**bold**](https://example.com)")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "bold" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "bold")
	}
	if len(b.Facets) != 2 {
		t.Fatalf("facets = %d, want 2", len(b.Facets))
	}
	types := map[string]bool{}
	for _, f := range b.Facets {
		types[f.Features[0].Type] = true
		if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
			t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
		}
	}
	if !types["pub.leaflet.richtext.facet#bold"] {
		t.Error("missing bold facet")
	}
	if !types["pub.leaflet.richtext.facet#link"] {
		t.Error("missing link facet")
	}
}

func TestCodeSpan(t *testing.T) {
	content, _ := parse("`code`")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "code" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "code")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	if b.Facets[0].Features[0].Type != "pub.leaflet.richtext.facet#code" {
		t.Errorf("feature = %q, want code", b.Facets[0].Features[0].Type)
	}
}

func TestStrikethrough(t *testing.T) {
	content, _ := parse("~~struck~~")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "struck" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "struck")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	if b.Facets[0].Features[0].Type != "pub.leaflet.richtext.facet#strikethrough" {
		t.Errorf("feature = %q, want strikethrough", b.Facets[0].Features[0].Type)
	}
}

func TestMultiByteUTF8(t *testing.T) {
	content, _ := parse("**café**")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "café" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "café")
	}
	// "café" is 5 bytes: c(1) a(1) f(1) é(2)
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 5 {
		t.Errorf("index = {%d,%d}, want {0,5}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

func TestPlainText(t *testing.T) {
	content, _ := parse("just plain text")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "just plain text" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "just plain text")
	}
	if len(b.Facets) != 0 {
		t.Errorf("facets = %d, want 0", len(b.Facets))
	}
}

func TestMixedInlines(t *testing.T) {
	content, _ := parse("hello **bold** and *italic* world")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "hello bold and italic world" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "hello bold and italic world")
	}
	if len(b.Facets) != 2 {
		t.Fatalf("facets = %d, want 2", len(b.Facets))
	}
}

// --- Block tests ---

func TestHeading(t *testing.T) {
	for _, tc := range []struct {
		md    string
		level int
		text  string
	}{
		{"# H1", 1, "H1"},
		{"## H2", 2, "H2"},
		{"### H3", 3, "H3"},
		{"###### H6", 6, "H6"},
	} {
		content, _ := parse(tc.md)
		b := getBlock(t, content, 0).(HeaderBlock)
		if b.Level != tc.level {
			t.Errorf("%q: level = %d, want %d", tc.md, b.Level, tc.level)
		}
		if b.Plaintext != tc.text {
			t.Errorf("%q: plaintext = %q, want %q", tc.md, b.Plaintext, tc.text)
		}
	}
}

func TestFencedCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(CodeBlock)

	if b.Language != "go" {
		t.Errorf("language = %q, want %q", b.Language, "go")
	}
	if b.Plaintext != "fmt.Println(\"hello\")" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "fmt.Println(\"hello\")")
	}
}

func TestBlockquoteSingleParagraph(t *testing.T) {
	content, _ := parse("> quoted text")
	b := getBlock(t, content, 0).(BlockquoteBlock)

	if b.Plaintext != "quoted text" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "quoted text")
	}
}

func TestBlockquoteMultiParagraph(t *testing.T) {
	md := "> first paragraph\n>\n> second paragraph"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(BlockquoteBlock)

	if b.Plaintext != "first paragraph\n\nsecond paragraph" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "first paragraph\n\nsecond paragraph")
	}
}

func TestHorizontalRule(t *testing.T) {
	content, _ := parse("---")
	b := getBlock(t, content, 0).(HorizontalRuleBlock)

	if b.Type != "pub.leaflet.blocks.horizontalRule" {
		t.Errorf("type = %q, want horizontalRule", b.Type)
	}
}

func TestUnorderedList(t *testing.T) {
	md := "- first\n- second\n- third"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(UnorderedListBlock)

	if len(b.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(b.Children))
	}
	texts := []string{"first", "second", "third"}
	for i, item := range b.Children {
		tb := item.Content.(TextBlock)
		if tb.Plaintext != texts[i] {
			t.Errorf("item %d: plaintext = %q, want %q", i, tb.Plaintext, texts[i])
		}
	}
}

func TestNestedList(t *testing.T) {
	md := "- parent\n  - child1\n  - child2"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(UnorderedListBlock)

	if len(b.Children) != 1 {
		t.Fatalf("top-level children = %d, want 1", len(b.Children))
	}
	parent := b.Children[0]
	tb := parent.Content.(TextBlock)
	if tb.Plaintext != "parent" {
		t.Errorf("parent plaintext = %q, want %q", tb.Plaintext, "parent")
	}
	if len(parent.Children) != 2 {
		t.Fatalf("nested children = %d, want 2", len(parent.Children))
	}
}

func TestMixedDocument(t *testing.T) {
	md := "# Title\n\nSome text.\n\n```\ncode\n```\n\n- item"
	content, _ := parse(md)

	blocks := content.Pages[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks = %d, want 4", len(blocks))
	}

	if _, ok := blocks[0].Block.(HeaderBlock); !ok {
		t.Error("block 0: expected HeaderBlock")
	}
	if _, ok := blocks[1].Block.(TextBlock); !ok {
		t.Error("block 1: expected TextBlock")
	}
	if _, ok := blocks[2].Block.(CodeBlock); !ok {
		t.Error("block 2: expected CodeBlock")
	}
	if _, ok := blocks[3].Block.(UnorderedListBlock); !ok {
		t.Error("block 3: expected UnorderedListBlock")
	}
}

func TestBlockquoteWithFormatting(t *testing.T) {
	md := "> **bold** in quote"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(BlockquoteBlock)

	if b.Plaintext != "bold in quote" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "bold in quote")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
		t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

// --- Additional facet edge case tests ---

func TestEmoji(t *testing.T) {
	// 🎉 is 4 bytes in UTF-8
	content, _ := parse("**🎉**")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "🎉" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "🎉")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
		t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

func TestCJKCharacters(t *testing.T) {
	// 日本語 is 9 bytes (3 bytes per character)
	content, _ := parse("**日本語**")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "日本語" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "日本語")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	if f.Index.ByteStart != 0 || f.Index.ByteEnd != 9 {
		t.Errorf("index = {%d,%d}, want {0,9}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

func TestFormattingAfterMultiByte(t *testing.T) {
	// "café **bold**" — "café " is 6 bytes (café=5, space=1), bold is 4
	content, _ := parse("café **bold**")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "café bold" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "café bold")
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	// "café " = 6 bytes, then "bold" = 4 bytes
	if f.Index.ByteStart != 6 || f.Index.ByteEnd != 10 {
		t.Errorf("index = {%d,%d}, want {6,10}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

func TestMixedInlinesByteOffsets(t *testing.T) {
	// "hello **bold** and *italic* world"
	// h(1)e(1)l(1)l(1)o(1) (1) = 6 bytes before "bold"
	// bold = 4 bytes, facet {6,10}
	// " and " = 5 bytes
	// italic = 6 bytes, facet {15,21}
	content, _ := parse("hello **bold** and *italic* world")
	b := getBlock(t, content, 0).(TextBlock)

	var boldFacet, italicFacet *Facet
	for i := range b.Facets {
		switch b.Facets[i].Features[0].Type {
		case "pub.leaflet.richtext.facet#bold":
			boldFacet = &b.Facets[i]
		case "pub.leaflet.richtext.facet#italic":
			italicFacet = &b.Facets[i]
		}
	}
	if boldFacet == nil {
		t.Fatal("missing bold facet")
	}
	if italicFacet == nil {
		t.Fatal("missing italic facet")
	}
	if boldFacet.Index.ByteStart != 6 || boldFacet.Index.ByteEnd != 10 {
		t.Errorf("bold index = {%d,%d}, want {6,10}", boldFacet.Index.ByteStart, boldFacet.Index.ByteEnd)
	}
	if italicFacet.Index.ByteStart != 15 || italicFacet.Index.ByteEnd != 21 {
		t.Errorf("italic index = {%d,%d}, want {15,21}", italicFacet.Index.ByteStart, italicFacet.Index.ByteEnd)
	}
}

func TestLinkWithFormattedText(t *testing.T) {
	content, _ := parse("[*italic* and **bold**](https://example.com)")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "italic and bold" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "italic and bold")
	}
	// 3 facets: italic{0,6}, bold{11,15}, link{0,15}
	if len(b.Facets) != 3 {
		t.Fatalf("facets = %d, want 3", len(b.Facets))
	}
	types := map[string]bool{}
	for _, f := range b.Facets {
		types[f.Features[0].Type] = true
	}
	if !types["pub.leaflet.richtext.facet#italic"] {
		t.Error("missing italic facet")
	}
	if !types["pub.leaflet.richtext.facet#bold"] {
		t.Error("missing bold facet")
	}
	if !types["pub.leaflet.richtext.facet#link"] {
		t.Error("missing link facet")
	}
}

func TestStrikethroughWithBold(t *testing.T) {
	content, _ := parse("~~**both**~~")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "both" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "both")
	}
	if len(b.Facets) != 2 {
		t.Fatalf("facets = %d, want 2", len(b.Facets))
	}
	types := map[string]bool{}
	for _, f := range b.Facets {
		types[f.Features[0].Type] = true
		if f.Index.ByteStart != 0 || f.Index.ByteEnd != 4 {
			t.Errorf("index = {%d,%d}, want {0,4}", f.Index.ByteStart, f.Index.ByteEnd)
		}
	}
	if !types["pub.leaflet.richtext.facet#bold"] {
		t.Error("missing bold facet")
	}
	if !types["pub.leaflet.richtext.facet#strikethrough"] {
		t.Error("missing strikethrough facet")
	}
}

func TestInlineImageAltText(t *testing.T) {
	content, _ := parse("before ![alt text](image.png) after")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "before alt text after" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "before alt text after")
	}
}

// --- Additional block tests ---

func TestIndentedCodeBlock(t *testing.T) {
	md := "    indented code\n    second line"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(CodeBlock)

	if b.Type != "pub.leaflet.blocks.code" {
		t.Errorf("type = %q, want code", b.Type)
	}
	if b.Language != "" {
		t.Errorf("language = %q, want empty", b.Language)
	}
}

func TestFencedCodeBlockNoLanguage(t *testing.T) {
	md := "```\nsome code\n```"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(CodeBlock)

	if b.Language != "" {
		t.Errorf("language = %q, want empty", b.Language)
	}
	if b.Plaintext != "some code" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "some code")
	}
}

func TestFencedCodeBlockMultiline(t *testing.T) {
	md := "```python\ndef hello():\n    print(\"hi\")\n```"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(CodeBlock)

	if b.Language != "python" {
		t.Errorf("language = %q, want %q", b.Language, "python")
	}
	want := "def hello():\n    print(\"hi\")"
	if b.Plaintext != want {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, want)
	}
}

func TestOrderedList(t *testing.T) {
	md := "1. first\n2. second\n3. third"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(OrderedListBlock)

	if b.Type != "pub.leaflet.blocks.orderedList" {
		t.Errorf("type = %q, want orderedList", b.Type)
	}
	if len(b.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(b.Children))
	}
	texts := []string{"first", "second", "third"}
	for i, item := range b.Children {
		tb := item.Content.(TextBlock)
		if tb.Plaintext != texts[i] {
			t.Errorf("item %d: plaintext = %q, want %q", i, tb.Plaintext, texts[i])
		}
	}
}

func TestOrderedListStartIndex(t *testing.T) {
	md := "3. third\n4. fourth"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(OrderedListBlock)

	if b.StartIndex != 3 {
		t.Errorf("startIndex = %d, want 3", b.StartIndex)
	}
}

func TestMixedNestedList_OrderedInsideUnordered(t *testing.T) {
	md := "- item\n  1. nested one\n  2. nested two"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(UnorderedListBlock)

	if len(b.Children) != 1 {
		t.Fatalf("top-level children = %d, want 1", len(b.Children))
	}
	item := b.Children[0]
	if item.OrderedListChildren == nil {
		t.Fatal("expected OrderedListChildren, got nil")
	}
	ol := item.OrderedListChildren
	if ol.Type != "pub.leaflet.blocks.orderedList" {
		t.Errorf("nested type = %q, want orderedList", ol.Type)
	}
	if len(ol.Children) != 2 {
		t.Fatalf("nested children = %d, want 2", len(ol.Children))
	}
}

func TestMixedNestedList_UnorderedInsideOrdered(t *testing.T) {
	md := "1. item\n   - nested a\n   - nested b"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(OrderedListBlock)

	if len(b.Children) != 1 {
		t.Fatalf("top-level children = %d, want 1", len(b.Children))
	}
	item := b.Children[0]
	if item.UnorderedListChildren == nil {
		t.Fatal("expected UnorderedListChildren, got nil")
	}
	ul := item.UnorderedListChildren
	if ul.Type != "pub.leaflet.blocks.unorderedList" {
		t.Errorf("nested type = %q, want unorderedList", ul.Type)
	}
	if len(ul.Children) != 2 {
		t.Fatalf("nested children = %d, want 2", len(ul.Children))
	}
}

func TestHardLineBreak(t *testing.T) {
	// Two spaces at end of line = hard line break in markdown
	content, _ := parse("line one  \nline two")
	b := getBlock(t, content, 0).(TextBlock)

	if b.Plaintext != "line one\nline two" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "line one\nline two")
	}
}

func TestListWithFormatting(t *testing.T) {
	md := "- **bold item**\n- *italic item*"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(UnorderedListBlock)

	if len(b.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(b.Children))
	}

	first := b.Children[0].Content.(TextBlock)
	if first.Plaintext != "bold item" {
		t.Errorf("item 0 plaintext = %q, want %q", first.Plaintext, "bold item")
	}
	if len(first.Facets) != 1 {
		t.Fatalf("item 0 facets = %d, want 1", len(first.Facets))
	}
	if first.Facets[0].Features[0].Type != "pub.leaflet.richtext.facet#bold" {
		t.Error("item 0: expected bold facet")
	}

	second := b.Children[1].Content.(TextBlock)
	if second.Plaintext != "italic item" {
		t.Errorf("item 1 plaintext = %q, want %q", second.Plaintext, "italic item")
	}
	if len(second.Facets) != 1 {
		t.Fatalf("item 1 facets = %d, want 1", len(second.Facets))
	}
	if second.Facets[0].Features[0].Type != "pub.leaflet.richtext.facet#italic" {
		t.Error("item 1: expected italic facet")
	}
}

func TestBlockquoteMultiParagraphFacetOffsets(t *testing.T) {
	md := "> **first**\n>\n> **second**"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(BlockquoteBlock)

	// "first\n\nsecond"
	// first: 5 bytes, bold {0,5}
	// \n\n: 2 bytes
	// second: 6 bytes, bold {7,13}
	if b.Plaintext != "first\n\nsecond" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "first\n\nsecond")
	}
	if len(b.Facets) != 2 {
		t.Fatalf("facets = %d, want 2", len(b.Facets))
	}
	f1 := b.Facets[0]
	if f1.Index.ByteStart != 0 || f1.Index.ByteEnd != 5 {
		t.Errorf("facet 0 index = {%d,%d}, want {0,5}", f1.Index.ByteStart, f1.Index.ByteEnd)
	}
	f2 := b.Facets[1]
	if f2.Index.ByteStart != 7 || f2.Index.ByteEnd != 13 {
		t.Errorf("facet 1 index = {%d,%d}, want {7,13}", f2.Index.ByteStart, f2.Index.ByteEnd)
	}
}

func TestHeadingWithFormatting(t *testing.T) {
	content, _ := parse("## Hello **world**")
	b := getBlock(t, content, 0).(HeaderBlock)

	if b.Plaintext != "Hello world" {
		t.Errorf("plaintext = %q, want %q", b.Plaintext, "Hello world")
	}
	if b.Level != 2 {
		t.Errorf("level = %d, want 2", b.Level)
	}
	if len(b.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(b.Facets))
	}
	f := b.Facets[0]
	// "Hello " = 6 bytes, "world" = 5 bytes
	if f.Index.ByteStart != 6 || f.Index.ByteEnd != 11 {
		t.Errorf("index = {%d,%d}, want {6,11}", f.Index.ByteStart, f.Index.ByteEnd)
	}
}

func TestEmptyDocument(t *testing.T) {
	content, _ := parse("")
	if len(content.Pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(content.Pages))
	}
	if len(content.Pages[0].Blocks) != 0 {
		t.Errorf("blocks = %d, want 0", len(content.Pages[0].Blocks))
	}
}

func TestHTMLBlockSkipped(t *testing.T) {
	md := "<div>html content</div>\n\nregular text"
	content, _ := parse(md)

	// HTML block should be skipped, only text block remains
	found := false
	for _, block := range content.Pages[0].Blocks {
		if tb, ok := block.Block.(TextBlock); ok && tb.Plaintext == "regular text" {
			found = true
		}
	}
	if !found {
		t.Error("expected text block 'regular text' after skipped HTML block")
	}
}

func TestNestedListThreeLevels(t *testing.T) {
	md := "- level1\n  - level2\n    - level3"
	content, _ := parse(md)
	b := getBlock(t, content, 0).(UnorderedListBlock)

	if len(b.Children) != 1 {
		t.Fatalf("top children = %d, want 1", len(b.Children))
	}
	l1 := b.Children[0]
	if l1.Content.(TextBlock).Plaintext != "level1" {
		t.Errorf("level1 = %q", l1.Content.(TextBlock).Plaintext)
	}
	if len(l1.Children) != 1 {
		t.Fatalf("level2 children = %d, want 1", len(l1.Children))
	}
	l2 := l1.Children[0]
	if l2.Content.(TextBlock).Plaintext != "level2" {
		t.Errorf("level2 = %q", l2.Content.(TextBlock).Plaintext)
	}
	if len(l2.Children) != 1 {
		t.Fatalf("level3 children = %d, want 1", len(l2.Children))
	}
	l3 := l2.Children[0]
	if l3.Content.(TextBlock).Plaintext != "level3" {
		t.Errorf("level3 = %q", l3.Content.(TextBlock).Plaintext)
	}
}

// --- JSON serialization tests ---

func TestContentJSON(t *testing.T) {
	content, _ := parse("# Hello\n\nworld")
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	pages, ok := m["pages"].([]any)
	if !ok || len(pages) != 1 {
		t.Fatalf("pages not found or wrong count")
	}
	page := pages[0].(map[string]any)
	if page["$type"] != "pub.leaflet.pages.linearDocument" {
		t.Errorf("page $type = %v", page["$type"])
	}

	blocks := page["blocks"].([]any)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(blocks))
	}

	// First block: header
	b0 := blocks[0].(map[string]any)["block"].(map[string]any)
	if b0["$type"] != "pub.leaflet.blocks.header" {
		t.Errorf("block 0 $type = %v", b0["$type"])
	}
	if b0["plaintext"] != "Hello" {
		t.Errorf("block 0 plaintext = %v", b0["plaintext"])
	}

	// Second block: text
	b1 := blocks[1].(map[string]any)["block"].(map[string]any)
	if b1["$type"] != "pub.leaflet.blocks.text" {
		t.Errorf("block 1 $type = %v", b1["$type"])
	}
	if b1["plaintext"] != "world" {
		t.Errorf("block 1 plaintext = %v", b1["plaintext"])
	}
}

func TestFacetJSONFormat(t *testing.T) {
	content, _ := parse("**bold**")
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)

	pages := m["pages"].([]any)
	blocks := pages[0].(map[string]any)["blocks"].([]any)
	block := blocks[0].(map[string]any)["block"].(map[string]any)
	facets := block["facets"].([]any)

	if len(facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(facets))
	}

	f := facets[0].(map[string]any)
	idx := f["index"].(map[string]any)
	if idx["byteStart"].(float64) != 0 {
		t.Errorf("byteStart = %v, want 0", idx["byteStart"])
	}
	if idx["byteEnd"].(float64) != 4 {
		t.Errorf("byteEnd = %v, want 4", idx["byteEnd"])
	}

	features := f["features"].([]any)
	feat := features[0].(map[string]any)
	if feat["$type"] != "pub.leaflet.richtext.facet#bold" {
		t.Errorf("feature $type = %v", feat["$type"])
	}
	// uri should be omitted for non-link facets
	if _, hasURI := feat["uri"]; hasURI {
		t.Error("non-link facet should not have uri field")
	}
}

func TestLinkFacetJSONHasURI(t *testing.T) {
	content, _ := parse("[text](https://example.com)")
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)

	pages := m["pages"].([]any)
	blocks := pages[0].(map[string]any)["blocks"].([]any)
	block := blocks[0].(map[string]any)["block"].(map[string]any)
	facets := block["facets"].([]any)
	feat := facets[0].(map[string]any)["features"].([]any)[0].(map[string]any)

	if feat["$type"] != "pub.leaflet.richtext.facet#link" {
		t.Errorf("feature $type = %v", feat["$type"])
	}
	if feat["uri"] != "https://example.com" {
		t.Errorf("uri = %v, want %q", feat["uri"], "https://example.com")
	}
}

func TestNoFacetsOmittedInJSON(t *testing.T) {
	content, _ := parse("plain text")
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	json.Unmarshal(data, &m)

	pages := m["pages"].([]any)
	blocks := pages[0].(map[string]any)["blocks"].([]any)
	block := blocks[0].(map[string]any)["block"].(map[string]any)

	if _, hasFacets := block["facets"]; hasFacets {
		t.Error("facets should be omitted from JSON when empty")
	}
}

// --- Footnote tests ---

func TestFootnoteSingle(t *testing.T) {
	md := "Text with a note.[^1]\n\n[^1]: The footnote definition."
	content, _ := parseWithFootnotes(md)

	blocks := content.Pages[0].Blocks
	// paragraph + hr + "Notes" header + footnote list = 4
	if len(blocks) != 4 {
		t.Fatalf("blocks = %d, want 4 (paragraph + hr + header + footnote list)", len(blocks))
	}
	if _, ok := blocks[1].Block.(HorizontalRuleBlock); !ok {
		t.Errorf("blocks[1] is not HorizontalRuleBlock")
	}
	hdr, ok := blocks[2].Block.(HeaderBlock)
	if !ok || hdr.Plaintext != "Notes" || hdr.Level != 2 {
		t.Errorf("blocks[2] = %+v, want HeaderBlock{Notes, level 2}", blocks[2].Block)
	}

	ul := blocks[3].Block.(UnorderedListBlock)
	if len(ul.Children) != 1 {
		t.Fatalf("footnote items = %d, want 1", len(ul.Children))
	}
	item := ul.Children[0].Content.(TextBlock)
	if item.Plaintext != "The footnote definition." {
		t.Errorf("plaintext = %q, want %q", item.Plaintext, "The footnote definition.")
	}
}

func TestFootnoteMultiple(t *testing.T) {
	md := "First[^a] and second[^b].\n\n[^a]: Alpha note.\n\n[^b]: Beta note."
	content, _ := parseWithFootnotes(md)

	blocks := content.Pages[0].Blocks
	ul := blocks[len(blocks)-1].Block.(UnorderedListBlock)

	if len(ul.Children) != 2 {
		t.Fatalf("footnote items = %d, want 2", len(ul.Children))
	}
	if ul.Children[0].Content.(TextBlock).Plaintext != "Alpha note." {
		t.Errorf("item 0 = %q", ul.Children[0].Content.(TextBlock).Plaintext)
	}
	if ul.Children[1].Content.(TextBlock).Plaintext != "Beta note." {
		t.Errorf("item 1 = %q", ul.Children[1].Content.(TextBlock).Plaintext)
	}
}

func TestFootnoteWithFormatting(t *testing.T) {
	md := "See note.[^1]\n\n[^1]: Visit [example.com](https://example.com) for details."
	content, _ := parseWithFootnotes(md)

	blocks := content.Pages[0].Blocks
	ul := blocks[len(blocks)-1].Block.(UnorderedListBlock)
	item := ul.Children[0].Content.(TextBlock)

	if item.Plaintext != "Visit example.com for details." {
		t.Errorf("plaintext = %q", item.Plaintext)
	}
	// Only the link facet; no #id anchor.
	if len(item.Facets) != 1 {
		t.Fatalf("facets = %d, want 1", len(item.Facets))
	}
	if item.Facets[0].Features[0].Type != "pub.leaflet.richtext.facet#link" {
		t.Errorf("facets[0] feature = %q, want #link", item.Facets[0].Features[0].Type)
	}
	if item.Facets[0].Features[0].URI != "https://example.com" {
		t.Errorf("uri = %q", item.Facets[0].Features[0].URI)
	}
}

func TestFootnoteMultiParagraph(t *testing.T) {
	md := "Note.[^1]\n\n[^1]: First paragraph.\n\n    Second paragraph."
	content, _ := parseWithFootnotes(md)

	blocks := content.Pages[0].Blocks
	ul := blocks[len(blocks)-1].Block.(UnorderedListBlock)
	item := ul.Children[0].Content.(TextBlock)

	if item.Plaintext != "First paragraph.\n\nSecond paragraph." {
		t.Errorf("plaintext = %q, want multi-paragraph joined with \\n\\n", item.Plaintext)
	}
}

func TestFootnoteInlineMarker(t *testing.T) {
	// The [^1] inline marker should appear as "[1]" plain text with no facet.
	md := "Before[^1] after.\n\n[^1]: Definition."
	content, _ := parseWithFootnotes(md)

	body := content.Pages[0].Blocks[0].Block.(TextBlock)
	if body.Plaintext != "Before[1] after." {
		t.Errorf("body plaintext = %q, want %q", body.Plaintext, "Before[1] after.")
	}
	if len(body.Facets) != 0 {
		t.Errorf("body facets = %d, want 0 (no link for fragment anchors)", len(body.Facets))
	}
}

func TestFootnoteIDFacet(t *testing.T) {
	// Footnote definition items should have no #id facet (fragment anchors not used).
	md := "First[^1] and second[^2].\n\n[^1]: Alpha.\n\n[^2]: Beta."
	content, _ := parseWithFootnotes(md)

	blocks := content.Pages[0].Blocks
	ul := blocks[len(blocks)-1].Block.(UnorderedListBlock)

	for i := range ul.Children {
		item := ul.Children[i].Content.(TextBlock)
		for _, f := range item.Facets {
			if f.Features[0].Type == "pub.leaflet.richtext.facet#id" {
				t.Errorf("item %d: unexpected #id facet", i)
			}
		}
	}
}

func TestFootnoteHTMLAnchors(t *testing.T) {
	// RenderHTML should render [1] as plain text and the definition as a plain <li>.
	md := "See note.[^1]\n\n[^1]: The definition."
	content, _ := parseWithFootnotes(md)
	out := RenderHTML(content)

	if !strings.Contains(out, `[1]`) {
		t.Errorf("missing inline marker [1]; got:\n%s", out)
	}
	if strings.Contains(out, `<a href="#fn-1">`) {
		t.Errorf("unexpected inline link to #fn-1; got:\n%s", out)
	}
	if strings.Contains(out, `id="fn-1"`) {
		t.Errorf("unexpected id anchor on footnote li; got:\n%s", out)
	}
}
