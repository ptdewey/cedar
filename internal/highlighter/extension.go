package highlighter

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type TreeSitterExtension struct{}

func NewTreeSitterExtension() goldmark.Extender {
	return &TreeSitterExtension{}
}

func (e *TreeSitterExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewTreeSitterRenderer(), 500),
		),
	)
}
