package leaflet

import (
	"strings"
	"testing"
)

func TestRenderRichText_Plain(t *testing.T) {
	got := renderRichText("hello world", nil)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestRenderRichText_Bold(t *testing.T) {
	got := renderRichText("bold text", []Facet{{
		Index:    ByteSlice{0, 4},
		Features: []Feature{{Type: "pub.leaflet.richtext.facet#bold"}},
	}})
	if got != "<strong>bold</strong> text" {
		t.Errorf("got %q", got)
	}
}

func TestRenderRichText_Link(t *testing.T) {
	got := renderRichText("click here", []Facet{{
		Index:    ByteSlice{0, 5},
		Features: []Feature{{Type: "pub.leaflet.richtext.facet#link", URI: "https://example.com"}},
	}})
	want := `<a href="https://example.com">click</a> here`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderRichText_BoldInsideLink(t *testing.T) {
	got := renderRichText("bold", []Facet{
		{Index: ByteSlice{0, 4}, Features: []Feature{{Type: "pub.leaflet.richtext.facet#bold"}}},
		{Index: ByteSlice{0, 4}, Features: []Feature{{Type: "pub.leaflet.richtext.facet#link", URI: "https://x.com"}}},
	})
	// Both bold and link should be present
	if !strings.Contains(got, "<strong>") || !strings.Contains(got, `<a href=`) {
		t.Errorf("got %q, expected bold+link", got)
	}
}

func TestRenderRichText_Strikethrough(t *testing.T) {
	got := renderRichText("struck", []Facet{{
		Index:    ByteSlice{0, 6},
		Features: []Feature{{Type: "pub.leaflet.richtext.facet#strikethrough"}},
	}})
	if got != "<s>struck</s>" {
		t.Errorf("got %q", got)
	}
}

func TestRenderRichText_Code(t *testing.T) {
	got := renderRichText("code", []Facet{{
		Index:    ByteSlice{0, 4},
		Features: []Feature{{Type: "pub.leaflet.richtext.facet#code"}},
	}})
	if got != "<code>code</code>" {
		t.Errorf("got %q", got)
	}
}

func TestRenderRichText_HTMLEscaping(t *testing.T) {
	got := renderRichText(`<script>alert("xss")</script>`, nil)
	if strings.Contains(got, "<script>") {
		t.Errorf("HTML not escaped: %q", got)
	}
}

func TestRenderHTML_FullDocument(t *testing.T) {
	content, _ := parse("# Hello\n\nSome **bold** text.\n\n```go\nfmt.Println()\n```\n\n---\n\n> quote\n\n- item one\n- item two\n")
	html := RenderHTML(content)

	checks := []string{
		"<h1>Hello</h1>",
		"<strong>bold</strong>",
		`<code class="language-go">`,
		"<hr>",
		"<blockquote>",
		"<ul>",
		"<li>item one</li>",
		"<li>item two</li>",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("missing %q in output", check)
		}
	}
}

func TestRenderHTML_MultiByteUTF8(t *testing.T) {
	content, _ := parse("**café**")
	html := RenderHTML(content)
	if !strings.Contains(html, "<strong>café</strong>") {
		t.Errorf("multi-byte not rendered correctly: %s", html)
	}
}

func TestRenderRichText_NewlineBecomesBreak(t *testing.T) {
	got := renderRichText("line one\nline two", nil)
	if got != "line one<br>line two" {
		t.Errorf("got %q, want %q", got, "line one<br>line two")
	}
}

func TestRenderHTML_OrderedList(t *testing.T) {
	content, _ := parse("1. first\n2. second\n3. third")
	h := RenderHTML(content)

	checks := []string{"<ol>", "<li>first</li>", "<li>second</li>", "<li>third</li>", "</ol>"}
	for _, check := range checks {
		if !strings.Contains(h, check) {
			t.Errorf("missing %q in output:\n%s", check, h)
		}
	}
}

func TestRenderHTML_OrderedListStartIndex(t *testing.T) {
	content, _ := parse("3. third\n4. fourth")
	h := RenderHTML(content)

	if !strings.Contains(h, `<ol start="3">`) {
		t.Errorf("missing start attribute in output:\n%s", h)
	}
}

func TestRenderHTML_HardLineBreak(t *testing.T) {
	content, _ := parse("line one  \nline two")
	h := RenderHTML(content)

	if !strings.Contains(h, "line one<br>line two") {
		t.Errorf("hard line break not rendered as <br>:\n%s", h)
	}
}

func TestComputeSegments_Overlapping(t *testing.T) {
	// "bold link" where bold covers 0-9 and link covers 5-9
	segments := computeSegments("bold link", []Facet{
		{Index: ByteSlice{0, 9}, Features: []Feature{{Type: "pub.leaflet.richtext.facet#bold"}}},
		{Index: ByteSlice{5, 9}, Features: []Feature{{Type: "pub.leaflet.richtext.facet#link", URI: "https://x.com"}}},
	})

	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}

	// First segment: "bold " - only bold
	if segments[0].text != "bold " {
		t.Errorf("segment 0 text: %q", segments[0].text)
	}
	if len(segments[0].features) != 1 {
		t.Errorf("segment 0 features: %d", len(segments[0].features))
	}

	// Second segment: "link" - bold + link
	if segments[1].text != "link" {
		t.Errorf("segment 1 text: %q", segments[1].text)
	}
	if len(segments[1].features) != 2 {
		t.Errorf("segment 1 features: %d", len(segments[1].features))
	}
}
