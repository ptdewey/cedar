package highlighter

import (
	"bytes"
	"context"
	"fmt"
	"io"

	stdhtml "html"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type TreeSitterRenderer struct{}

func NewTreeSitterRenderer() renderer.NodeRenderer {
	return &TreeSitterRenderer{}
}

func (r *TreeSitterRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *TreeSitterRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	codeBlock := node.(*ast.FencedCodeBlock)
	lang := string(codeBlock.Language(source))
	var codeBuf bytes.Buffer
	for i := 0; i < codeBlock.Lines().Len(); i++ {
		segment := codeBlock.Lines().At(i)
		codeBuf.Write(segment.Value(source))
	}
	code := codeBuf.String()

	htmlStr, err := Highlight(context.Background(), code, lang)
	if err != nil {
		// fallback: plain HTML-escaped code
		htmlStr = fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>", lang, stdhtml.EscapeString(code))
	}
	_, err = io.WriteString(w, htmlStr)
	return ast.WalkContinue, err
}

// TODO: add options for compat
func (r *TreeSitterRenderer) AddOptions(opts ...html.Option) {}
