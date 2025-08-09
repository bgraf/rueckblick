package render

import (
	"path/filepath"

	"github.com/bgraf/rueckblick/data"
)

// postprocessDocument modifies the rendered document by replacing links, image and video sources.
func Render(doc *data.Document, opts data.StoreOptions) {
	ImplicitFigure(doc)

	toResource := func(original string) (data.Resource, bool) {
		srcPath := original
		if !filepath.IsAbs(original) {
			srcPath = filepath.Join(filepath.Dir(doc.Path), original)
		}
		resource, _ := opts.RenderImagePath(doc, srcPath)

		return resource, true
	}

	RecodePaths(doc, toResource)

	// Must be executed in this order, because GPX requires populated galleries.
	EmplaceGalleries(doc, toResource)
	EmplaceGPXMaps(doc, toResource)
	EmplaceVideos(doc, toResource)

	doc.IsHtmlProcessed = true
}
