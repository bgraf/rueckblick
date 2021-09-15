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
	resPath, ok := g.options.ProvideSource(node.mapNo, gpxFilePath)
	if !ok {
		return ast.WalkSkipChildren, nil
	}

	_, _ = w.WriteString("<div class=\"gpx-map\" id=\"")
	_, _ = w.WriteString(ElementID(node.mapNo))
	_, _ = w.WriteString("\">")

	_, _ = w.WriteString(fmt.Sprintf(`
		<script>
		(function () {
			let mapContainer = document.currentScript.parentElement;
			window.addEventListener('DOMContentLoaded', function() {
				loadAndMountMap(mapContainer, { 'dataURL': '%s' })
			});
		})();
		</script>`,
		resPath,
	))
	_, _ = w.WriteString("</div>")

	return ast.WalkSkipChildren, nil
}
