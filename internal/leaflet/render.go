package leaflet

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

// RenderHTML renders a Leaflet Document as a standalone HTML page for local preview.
func RenderHTML(content *Document) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Leaflet Preview</title>
<style>
  body { max-width: 700px; margin: 2rem auto; padding: 0 1rem; font-family: system-ui, sans-serif; line-height: 1.6; color: #222; }
  a { color: #0055aa; }
  code { font-size: 0.9em; background: #f0f0f0; padding: 0.1em 0.3em; border-radius: 3px; }
  pre { background: #f5f5f5; padding: 1rem; overflow-x: auto; border-radius: 4px; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 3px solid #ccc; margin-left: 0; padding-left: 1rem; color: #555; }
  hr { border: none; border-top: 1px solid #ddd; margin: 2rem 0; }
  ul { padding-left: 1.5rem; }
  li { margin-bottom: 0.25rem; }
  h1 { font-size: 2em; margin: 0.67em 0; }
  h2 { font-size: 1.5em; margin: 0.75em 0; }
  h3 { font-size: 1.17em; margin: 0.83em 0; }
  h4, h5, h6 { font-size: 1em; margin: 1em 0; }
  p { margin: 0.5em 0; }
</style>
</head>
<body>
`)

	for _, page := range content.Pages {
		for _, entry := range page.Blocks {
			renderBlock(&sb, entry.Block)
		}
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

func renderBlock(sb *strings.Builder, block any) {
	switch b := block.(type) {
	case TextBlock:
		sb.WriteString("<p>")
		sb.WriteString(renderRichText(b.Plaintext, b.Facets))
		sb.WriteString("</p>\n")

	case HeaderBlock:
		level := b.Level
		// Leaflet renders level 1 as h2, level 2 as h3, etc.
		tag := fmt.Sprintf("h%d", level)
		if level > 6 {
			tag = "h6"
		}
		sb.WriteString("<" + tag + ">")
		sb.WriteString(renderRichText(b.Plaintext, b.Facets))
		sb.WriteString("</" + tag + ">\n")

	case CodeBlock:
		sb.WriteString("<pre>")
		if b.Language != "" {
			fmt.Fprintf(sb, `<code class="language-%s">`, html.EscapeString(b.Language))
		} else {
			sb.WriteString("<code>")
		}
		sb.WriteString(html.EscapeString(b.Plaintext))
		sb.WriteString("</code></pre>\n")

	case BlockquoteBlock:
		sb.WriteString("<blockquote>")
		// Split on \n\n for multi-paragraph blockquotes
		paras := strings.Split(b.Plaintext, "\n\n")
		if len(paras) == 1 {
			sb.WriteString("<p>")
			sb.WriteString(renderRichText(b.Plaintext, b.Facets))
			sb.WriteString("</p>")
		} else {
			// Render each paragraph, splitting facets by offset
			offset := 0
			for i, para := range paras {
				if i > 0 {
					offset += 2 // "\n\n"
				}
				paraEnd := offset + len(para)
				var paraFacets []Facet
				for _, f := range b.Facets {
					if f.Index.ByteStart >= offset && f.Index.ByteEnd <= paraEnd {
						paraFacets = append(paraFacets, Facet{
							Index: ByteSlice{
								ByteStart: f.Index.ByteStart - offset,
								ByteEnd:   f.Index.ByteEnd - offset,
							},
							Features: f.Features,
						})
					}
				}
				sb.WriteString("<p>")
				sb.WriteString(renderRichText(para, paraFacets))
				sb.WriteString("</p>")
				offset = paraEnd
			}
		}
		sb.WriteString("</blockquote>\n")

	case HorizontalRuleBlock:
		sb.WriteString("<hr>\n")

	case UnorderedListBlock:
		sb.WriteString("<ul>\n")
		for _, item := range b.Children {
			renderListItem(sb, item)
		}
		sb.WriteString("</ul>\n")

	case OrderedListBlock:
		writeOLOpen(sb, b.StartIndex)
		for _, item := range b.Children {
			renderListItem(sb, item)
		}
		sb.WriteString("</ol>\n")
	}
}

func writeOLOpen(sb *strings.Builder, startIndex int) {
	if startIndex > 1 {
		fmt.Fprintf(sb, "<ol start=\"%d\">\n", startIndex)
	} else {
		sb.WriteString("<ol>\n")
	}
}

func renderListItem(sb *strings.Builder, item ListItem) {
	sb.WriteString("<li>")
	if item.Content != nil {
		switch c := item.Content.(type) {
		case TextBlock:
			sb.WriteString(renderRichText(c.Plaintext, c.Facets))
		case HeaderBlock:
			sb.WriteString(renderRichText(c.Plaintext, c.Facets))
		}
	}
	if len(item.Children) > 0 {
		sb.WriteString("\n<ul>\n")
		for _, child := range item.Children {
			renderListItem(sb, child)
		}
		sb.WriteString("</ul>")
	}
	if item.OrderedListChildren != nil {
		ol := item.OrderedListChildren
		sb.WriteByte('\n')
		writeOLOpen(sb, ol.StartIndex)
		for _, child := range ol.Children {
			renderListItem(sb, child)
		}
		sb.WriteString("</ol>")
	}
	if item.UnorderedListChildren != nil {
		sb.WriteString("\n<ul>\n")
		for _, child := range item.UnorderedListChildren.Children {
			renderListItem(sb, child)
		}
		sb.WriteString("</ul>")
	}
	sb.WriteString("</li>\n")
}

// segment represents a non-overlapping text segment with merged facet features.
type segment struct {
	text     string
	features []Feature
}

// renderRichText implements the same algorithm as Leaflet's RichText.segments()
// generator, producing HTML with appropriate tags for facet features.
func renderRichText(plaintext string, facets []Facet) string {
	if len(facets) == 0 {
		return strings.ReplaceAll(html.EscapeString(plaintext), "\n", "<br>")
	}

	segments := computeSegments(plaintext, facets)

	var sb strings.Builder
	for _, seg := range segments {
		if len(seg.features) == 0 {
			sb.WriteString(strings.ReplaceAll(html.EscapeString(seg.text), "\n", "<br>"))
			continue
		}

		escaped := strings.ReplaceAll(html.EscapeString(seg.text), "\n", "<br>")

		// Find link feature if any
		var linkURI string
		for _, f := range seg.features {
			if f.Type == "pub.leaflet.richtext.facet#link" {
				linkURI = f.URI
				break
			}
		}

		// Build inline styling
		var isCode, isBold, isItalic, isStrikethrough bool
		for _, f := range seg.features {
			switch f.Type {
			case "pub.leaflet.richtext.facet#bold":
				isBold = true
			case "pub.leaflet.richtext.facet#italic":
				isItalic = true
			case "pub.leaflet.richtext.facet#code":
				isCode = true
			case "pub.leaflet.richtext.facet#strikethrough":
				isStrikethrough = true
			}
		}

		text := escaped
		if isCode {
			text = "<code>" + text + "</code>"
		}
		if isBold {
			text = "<strong>" + text + "</strong>"
		}
		if isItalic {
			text = "<em>" + text + "</em>"
		}
		if isStrikethrough {
			text = "<s>" + text + "</s>"
		}
		if linkURI != "" {
			text = fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(linkURI), text)
		}

		sb.WriteString(text)
	}

	return sb.String()
}

// computeSegments splits plaintext into non-overlapping segments with merged
// features, mirroring Leaflet's RichText.segments() algorithm.
func computeSegments(plaintext string, facets []Facet) []segment {
	if len(facets) == 0 {
		return []segment{{text: plaintext}}
	}

	// Sort facets by byteStart
	sorted := make([]Facet, len(facets))
	copy(sorted, facets)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Index.ByteStart < sorted[j].Index.ByteStart
	})

	// Collect all boundary points
	pointSet := make(map[int]bool)
	pointSet[0] = true
	pointSet[len(plaintext)] = true
	for _, f := range sorted {
		if f.Index.ByteStart >= 0 && f.Index.ByteStart <= len(plaintext) {
			pointSet[f.Index.ByteStart] = true
		}
		if f.Index.ByteEnd >= 0 && f.Index.ByteEnd <= len(plaintext) {
			pointSet[f.Index.ByteEnd] = true
		}
	}

	points := make([]int, 0, len(pointSet))
	for p := range pointSet {
		points = append(points, p)
	}
	sort.Ints(points)

	// Build segments between consecutive boundary points
	var segments []segment
	for i := 0; i < len(points)-1; i++ {
		start := points[i]
		end := points[i+1]
		if start == end {
			continue
		}

		// Collect all features active at this range
		var features []Feature
		for _, f := range sorted {
			if f.Index.ByteStart <= start && f.Index.ByteEnd >= end {
				features = append(features, f.Features...)
			}
		}

		segments = append(segments, segment{
			text:     plaintext[start:end],
			features: features,
		})
	}

	return segments
}
