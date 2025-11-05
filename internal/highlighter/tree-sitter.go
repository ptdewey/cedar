package highlighter

import (
	"context"
	"fmt"
	"html"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tsbash "github.com/smacker/go-tree-sitter/bash"
	tsgo "github.com/smacker/go-tree-sitter/golang"
	tslua "github.com/smacker/go-tree-sitter/lua"
)

var langs = map[string]*sitter.Language{
	"go":   tsgo.GetLanguage(),
	"bash": tsbash.GetLanguage(),
	"sh":   tsbash.GetLanguage(),
	"lua":  tslua.GetLanguage(),
}

func Highlight(ctx context.Context, code, lang string) (string, error) {
	langParser, ok := langs[strings.ToLower(lang)]
	if !ok {
		return html.EscapeString(code), fmt.Errorf("unsupported language: %s", lang)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(langParser)

	tree, err := parser.ParseCtx(ctx, nil, []byte(code))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<pre><code class="language-` + lang + `">`)
	highlightRecursive(&b, code, tree.RootNode(), lang)
	b.WriteString(`</code></pre>`)

	return b.String(), nil
}

func highlightRecursive(b *strings.Builder, src string, node *sitter.Node, lang string) {
	if node.ChildCount() == 0 {
		writeNodeText(b, src, node, lang)
		return
	}

	start := node.StartByte()
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		// write any text before this child
		if gap := child.StartByte() - start; gap > 0 {
			b.WriteString(html.EscapeString(src[start:child.StartByte()]))
		}
		highlightRecursive(b, src, child, lang)
		start = child.EndByte()
	}
	// write trailing text
	if end := node.EndByte(); end > start {
		b.WriteString(html.EscapeString(src[start:end]))
	}
}

func writeNodeText(b *strings.Builder, src string, node *sitter.Node, lang string) {
	// TODO: replace `node.Type` with a node classifier function to map semantic classes
	class := node.Type()
	text := html.EscapeString(src[node.StartByte():node.EndByte()])
	if class != "" {
		fmt.Fprintf(b, `<span class="tok-%s">%s</span>`, class, text)
	} else {
		b.WriteString(text)
	}
}
