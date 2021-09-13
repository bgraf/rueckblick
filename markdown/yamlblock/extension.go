package yamlblock

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type yamlBlock struct {
	addinByKey map[string]Addin
}

func New(addins ...Addin) *yamlBlock {
	addinByKey := make(map[string]Addin)
	for _, addin := range addins {
		key := addin.AddinKey()
		addinByKey[key] = addin
	}

	return &yamlBlock{
		addinByKey: addinByKey,
	}
}

func (y *yamlBlock) findAddin(key string) Addin {
	addin, ok := y.addinByKey[key]
	if !ok {
		return nil
	}

	return addin
}

func (y *yamlBlock) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(
				newYamlBlockParser(y),
				999,
			),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(newYamlBlockRenderer(), 500),
		),
	)
}

type Addin interface {
	AddinKey() string
	Make(pc parser.Context) interface{}
	Render(w util.BufWriter, source []byte, node interface{}, entering bool) (ast.WalkStatus, error)
}
