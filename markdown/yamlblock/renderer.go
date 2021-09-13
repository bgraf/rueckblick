package yamlblock

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type yamlBlockRenderer struct {
}

func newYamlBlockRenderer() renderer.NodeRenderer {
	return &yamlBlockRenderer{}
}

func (r *yamlBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(yamlBlockParserNodeKind, r.renderProxy)
}

func (r *yamlBlockRenderer) renderProxy(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	node := n.(*yamlBlockParserNode)
	if node.error != nil {
		// Some error happened during parsing, we forward it now and stop the rendering.
		return ast.WalkStop, node.error
	}

	return node.addin.Render(w, source, node.object, entering)
}
