package leaflet

import (
	"bytes"
	"strings"

	"github.com/google/uuid"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// blockEntry is the $type value for every entry in a linearDocument's blocks array.
const blockEntry = "pub.leaflet.pages.linearDocument#block"

// Convert transforms a goldmark AST document into a Leaflet document record.
func Convert(source []byte, doc ast.Node) *Document {
	var blocks []BlockEntry

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		blocks = append(blocks, convertBlock(source, child)...)
	}

	pageID, _ := uuid.NewV7()
	return &Document{
		Type: "pub.leaflet.content",
		Pages: []LinearDocument{{
			Type:   "pub.leaflet.pages.linearDocument",
			ID:     pageID.String(),
			Blocks: blocks,
		}},
	}
}

func convertBlock(source []byte, node ast.Node) []BlockEntry {
	switch n := node.(type) {
	case *ast.Paragraph:
		text, facets := flattenInlines(n, source)
		return []BlockEntry{{Type: blockEntry, Block: TextBlock{
			Type:      "pub.leaflet.blocks.text",
			Plaintext: text,
			Facets:    facets,
		}}}

	case *ast.Heading:
		text, facets := flattenInlines(n, source)
		return []BlockEntry{{Type: blockEntry, Block: HeaderBlock{
			Type:      "pub.leaflet.blocks.header",
			Plaintext: text,
			Level:     n.Level,
			Facets:    facets,
		}}}

	case *ast.FencedCodeBlock:
		return []BlockEntry{{Type: blockEntry, Block: CodeBlock{
			Type:      "pub.leaflet.blocks.code",
			Plaintext: linesText(source, n),
			Language:  string(n.Language(source)),
		}}}

	case *ast.CodeBlock:
		return []BlockEntry{{Type: blockEntry, Block: CodeBlock{
			Type:      "pub.leaflet.blocks.code",
			Plaintext: linesText(source, n),
		}}}

	case *ast.Blockquote:
		text, facets := flattenBlockquote(source, n)
		return []BlockEntry{{Type: blockEntry, Block: BlockquoteBlock{
			Type:      "pub.leaflet.blocks.blockquote",
			Plaintext: text,
			Facets:    facets,
		}}}

	case *ast.ThematicBreak:
		return []BlockEntry{{Type: blockEntry, Block: HorizontalRuleBlock{
			Type: "pub.leaflet.blocks.horizontalRule",
		}}}

	case *ast.List:
		if n.IsOrdered() {
			return []BlockEntry{{Type: blockEntry, Block: convertOrderedList(source, n)}}
		}
		return []BlockEntry{{Type: blockEntry, Block: convertUnorderedList(source, n)}}

	case *ast.HTMLBlock:
		return nil

	case *extast.FootnoteList:
		return []BlockEntry{
			{Type: blockEntry, Block: HorizontalRuleBlock{Type: "pub.leaflet.blocks.horizontalRule"}},
			{Type: blockEntry, Block: HeaderBlock{
				Type:      "pub.leaflet.blocks.header",
				Plaintext: "Notes",
				Level:     2,
			}},
			{Type: blockEntry, Block: convertFootnoteList(source, n)},
		}

	default:
		return nil
	}
}

// linesText concatenates all line segments of a block node into a string.
func linesText(source []byte, node ast.Node) string {
	var buf bytes.Buffer
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(source))
	}
	return strings.TrimRight(buf.String(), "\n")
}

// flattenBlockquote flattens all child paragraphs of a blockquote into
// a single plaintext string with merged facets.
func flattenBlockquote(source []byte, bq *ast.Blockquote) (string, []Facet) {
	return flattenParagraphChildren(source, bq)
}

// flattenParagraphChildren iterates the direct children of node, collects
// text from *ast.Paragraph children (skipping everything else), and joins
// them with "\n\n" while adjusting facet byte offsets accordingly.
func flattenParagraphChildren(source []byte, node ast.Node) (string, []Facet) {
	var parts []string
	var allFacets []Facet
	var offset int

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.Paragraph); !ok {
			continue
		}
		text, facets := flattenInlines(child, source)
		if len(parts) > 0 {
			offset += 2 // for "\n\n" separator
		}
		for _, f := range facets {
			allFacets = append(allFacets, Facet{
				Index: ByteSlice{
					ByteStart: f.Index.ByteStart + offset,
					ByteEnd:   f.Index.ByteEnd + offset,
				},
				Features: f.Features,
			})
		}
		offset += len(text)
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n"), allFacets
}

// convertUnorderedList converts a goldmark List node to an UnorderedListBlock.
func convertUnorderedList(source []byte, list *ast.List) UnorderedListBlock {
	var items []ListItem
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if li, ok := child.(*ast.ListItem); ok {
			items = append(items, convertListItem(source, li, false))
		}
	}
	return UnorderedListBlock{
		Type:     "pub.leaflet.blocks.unorderedList",
		Children: items,
	}
}

// convertOrderedList converts a goldmark List node to an OrderedListBlock.
func convertOrderedList(source []byte, list *ast.List) OrderedListBlock {
	var items []ListItem
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if li, ok := child.(*ast.ListItem); ok {
			items = append(items, convertListItem(source, li, true))
		}
	}
	b := OrderedListBlock{
		Type:     "pub.leaflet.blocks.orderedList",
		Children: items,
	}
	if list.Start > 1 {
		b.StartIndex = list.Start
	}
	return b
}

// convertListItem converts a single list item.
// parentOrdered indicates whether this item lives inside an ordered list,
// which determines how a cross-type nested list is stored.
func convertListItem(source []byte, li *ast.ListItem, parentOrdered bool) ListItem {
	listItemType := "pub.leaflet.blocks.unorderedList#listItem"
	if parentOrdered {
		listItemType = "pub.leaflet.blocks.orderedList#listItem"
	}
	item := ListItem{Type: listItemType}

	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			if item.Content == nil {
				text, facets := flattenInlines(n, source)
				item.Content = TextBlock{
					Type:      "pub.leaflet.blocks.text",
					Plaintext: text,
					Facets:    facets,
				}
			}
		case *ast.List:
			if n.IsOrdered() {
				if parentOrdered {
					// ordered inside ordered → same-type children
					nested := convertOrderedList(source, n)
					item.Children = append(item.Children, nested.Children...)
				} else {
					// ordered inside unordered → cross-type field
					nested := convertOrderedList(source, n)
					item.OrderedListChildren = &nested
				}
			} else {
				if !parentOrdered {
					// unordered inside unordered → same-type children
					nested := convertUnorderedList(source, n)
					item.Children = append(item.Children, nested.Children...)
				} else {
					// unordered inside ordered → cross-type field
					nested := convertUnorderedList(source, n)
					item.UnorderedListChildren = &nested
				}
			}
		}
	}

	if item.Content == nil {
		item.Content = TextBlock{
			Type:      "pub.leaflet.blocks.text",
			Plaintext: "",
		}
	}

	return item
}

// convertFootnoteList converts a goldmark FootnoteList node into an
// UnorderedListBlock, one item per footnote definition.
func convertFootnoteList(source []byte, fl *extast.FootnoteList) UnorderedListBlock {
	var items []ListItem
	for child := fl.FirstChild(); child != nil; child = child.NextSibling() {
		fn, ok := child.(*extast.Footnote)
		if !ok {
			continue
		}
		items = append(items, convertFootnote(source, fn))
	}
	return UnorderedListBlock{
		Type:     "pub.leaflet.blocks.unorderedList",
		Children: items,
	}
}

// convertFootnote converts a single Footnote node to a ListItem.
// Multi-paragraph footnotes are joined with \n\n, matching the blockquote
// flattening strategy. FootnoteBacklink nodes (the ↩ markers) are
// automatically ignored because they have no children and fall through the
// default case in walkInlines.
func convertFootnote(source []byte, fn *extast.Footnote) ListItem {
	text, facets := flattenParagraphChildren(source, fn)
	return ListItem{
		Content: TextBlock{
			Type:      "pub.leaflet.blocks.text",
			Plaintext: text,
			Facets:    facets,
		},
	}
}
