package gpx

import (
	"fmt"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

func (g *GPXAddin) Render(w util.BufWriter, source []byte, object interface{}, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	node := object.(*gpxNode)
	gpxFilePath := filepath.Join(filepath.Dir(node.documentPath), node.File)
	resPath, ok := g.sourceProvider.ProvideGPXSource(gpxFilePath)
	if !ok {
		return ast.WalkSkipChildren, nil
	}

	w.WriteString("<div class=\"gpx-map\" id=\"")
	w.WriteString(ElementID(node.count))
	w.WriteString("\">")

	w.WriteString(fmt.Sprintf(`
		<script>
		let mapContainer = document.currentScript.parentElement;
		window.addEventListener('DOMContentLoaded', function() {
			loadAndMountMap(mapContainer, { 'dataURL': '%s' })
		});
		</script>`,
		resPath,
	))
	w.WriteString("</div>")

	return ast.WalkSkipChildren, nil
}
