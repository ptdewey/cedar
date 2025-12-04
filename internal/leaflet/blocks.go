package leaflet

// Document is the pub.leaflet.content value embedded in a site.standard.document record.
type Document struct {
	Type  string           `json:"$type"`
	Pages []LinearDocument `json:"pages"`
}

// LinearDocument represents a pub.leaflet.pages.linearDocument page.
type LinearDocument struct {
	Type   string       `json:"$type"`
	ID     string       `json:"id,omitempty"`
	Blocks []BlockEntry `json:"blocks"`
}

// BlockEntry wraps a block in a linearDocument's blocks array.
type BlockEntry struct {
	Type  string `json:"$type"`
	Block any    `json:"block"`
}

// TextBlock represents a pub.leaflet.blocks.text block.
type TextBlock struct {
	Type      string  `json:"$type"`
	Plaintext string  `json:"plaintext"`
	Facets    []Facet `json:"facets,omitempty"`
}

// HeaderBlock represents a pub.leaflet.blocks.header block.
type HeaderBlock struct {
	Type      string  `json:"$type"`
	Plaintext string  `json:"plaintext"`
	Level     int     `json:"level"`
	Facets    []Facet `json:"facets,omitempty"`
}

// CodeBlock represents a pub.leaflet.blocks.code block.
type CodeBlock struct {
	Type      string `json:"$type"`
	Plaintext string `json:"plaintext"`
	Language  string `json:"language,omitempty"`
}

// BlockquoteBlock represents a pub.leaflet.blocks.blockquote block.
type BlockquoteBlock struct {
	Type      string  `json:"$type"`
	Plaintext string  `json:"plaintext"`
	Facets    []Facet `json:"facets,omitempty"`
}

// HorizontalRuleBlock represents a pub.leaflet.blocks.horizontalRule block.
type HorizontalRuleBlock struct {
	Type string `json:"$type"`
}

// UnorderedListBlock represents a pub.leaflet.blocks.unorderedList block.
type UnorderedListBlock struct {
	Type     string     `json:"$type"`
	Children []ListItem `json:"children"`
}

// OrderedListBlock represents a pub.leaflet.blocks.orderedList block.
type OrderedListBlock struct {
	Type       string     `json:"$type"`
	StartIndex int        `json:"startIndex,omitempty"`
	Children   []ListItem `json:"children"`
}

// ListItem represents a single item in an ordered or unordered list.
// Children holds nested items of the same list type.
// OrderedListChildren and UnorderedListChildren hold a nested list of the
// opposite type (mutually exclusive with each other).
type ListItem struct {
	Type                  string              `json:"$type"`
	Content               any                 `json:"content"`
	Children              []ListItem          `json:"children,omitempty"`
	OrderedListChildren   *OrderedListBlock   `json:"orderedListChildren,omitempty"`
	UnorderedListChildren *UnorderedListBlock `json:"unorderedListChildren,omitempty"`
}

// Facet represents a pub.leaflet.richtext.facet annotation.
type Facet struct {
	Index    ByteSlice `json:"index"`
	Features []Feature `json:"features"`
}

// ByteSlice represents a UTF-8 byte range within plain text.
type ByteSlice struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

// Feature represents a single facet feature (bold, italic, link, etc.).
type Feature struct {
	Type string `json:"$type"`
	URI  string `json:"uri,omitempty"`
}
