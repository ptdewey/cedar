package leaflet

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

type openFacet struct {
	byteStart int
	feature   Feature
}

// flattenInlines walks the inline children of a block node and produces
// plain text with byte-offset facets for rich text formatting.
func flattenInlines(node ast.Node, source []byte) (string, []Facet) {
	var buf bytes.Buffer
	var facets []Facet
	var stack []openFacet

	walkInlines(&buf, &facets, &stack, node, source)

	return buf.String(), facets
}

func walkInlines(buf *bytes.Buffer, facets *[]Facet, stack *[]openFacet, node ast.Node, source []byte) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			buf.Write(n.Value(source))
			if n.HardLineBreak() {
				buf.WriteByte('\n')
			} else if n.SoftLineBreak() {
				buf.WriteByte(' ')
			}

		case *ast.String:
			buf.Write(n.Value)

		case *ast.Emphasis:
			start := buf.Len()
			var feature Feature
			if n.Level == 2 {
				feature = Feature{Type: "pub.leaflet.richtext.facet#bold"}
			} else {
				feature = Feature{Type: "pub.leaflet.richtext.facet#italic"}
			}
			*stack = append(*stack, openFacet{byteStart: start, feature: feature})
			walkInlines(buf, facets, stack, n, source)
			open := (*stack)[len(*stack)-1]
			*stack = (*stack)[:len(*stack)-1]
			if buf.Len() > open.byteStart {
				*facets = append(*facets, Facet{
					Index:    ByteSlice{ByteStart: open.byteStart, ByteEnd: buf.Len()},
					Features: []Feature{open.feature},
				})
			}

		case *ast.Link:
			start := buf.Len()
			walkInlines(buf, facets, stack, n, source)
			if buf.Len() > start {
				*facets = append(*facets, Facet{
					Index: ByteSlice{ByteStart: start, ByteEnd: buf.Len()},
					Features: []Feature{{
						Type: "pub.leaflet.richtext.facet#link",
						URI:  string(n.Destination),
					}},
				})
			}

		case *ast.AutoLink:
			start := buf.Len()
			label := n.Label(source)
			buf.Write(label)
			*facets = append(*facets, Facet{
				Index: ByteSlice{ByteStart: start, ByteEnd: buf.Len()},
				Features: []Feature{{
					Type: "pub.leaflet.richtext.facet#link",
					URI:  string(n.URL(source)),
				}},
			})

		case *ast.CodeSpan:
			start := buf.Len()
			// Code spans contain Text children with the code content
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					buf.Write(t.Value(source))
					if t.SoftLineBreak() {
						buf.WriteByte(' ')
					}
				}
			}
			if buf.Len() > start {
				*facets = append(*facets, Facet{
					Index:    ByteSlice{ByteStart: start, ByteEnd: buf.Len()},
					Features: []Feature{{Type: "pub.leaflet.richtext.facet#code"}},
				})
			}

		case *extast.Strikethrough:
			start := buf.Len()
			walkInlines(buf, facets, stack, n, source)
			if buf.Len() > start {
				*facets = append(*facets, Facet{
					Index:    ByteSlice{ByteStart: start, ByteEnd: buf.Len()},
					Features: []Feature{{Type: "pub.leaflet.richtext.facet#strikethrough"}},
				})
			}

		case *ast.Image:
			// Lossy: just emit alt text, no image block
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					buf.Write(t.Value(source))
				}
			}

		case *ast.RawHTML:
			// Skip inline HTML

		case *extast.FootnoteLink:
			// Emit as plain text — fragment links (#fn-N) don't work in Leaflet.
			fmt.Fprintf(buf, "[%d]", n.Index)

		default:
			// Unknown inline — recurse into children
			walkInlines(buf, facets, stack, child, source)
		}
	}
}
