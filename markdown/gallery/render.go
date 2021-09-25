package gallery

import (
	"sort"
	"strings"
	"time"

	"log"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"

	"github.com/bgraf/rueckblick/images"
)

func (g *GalleryAddin) Render(w util.BufWriter, source []byte, object interface{}, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}

	node := object.(*galleryNode)
	addDefaultValues(node)

	files, err := node.findImagePaths()
	if err != nil {
		return ast.WalkStop, err
	}

	filesExif := make(
		[]struct {
			path string
			exif *images.EXIFData
		},
		len(files),
	)

	for i, filePath := range files {
		filesExif[i].path = filePath

		var err error

		filesExif[i].exif, err = images.ReadEXIFFromFile(filePath)
		if err != nil {
			log.Printf("could not load exif data: %s", err)
		}
	}

	sort.Slice(
		filesExif,
		func(i, j int) bool {
			f1 := filesExif[i]
			f2 := filesExif[j]

			if f1.exif == nil && f2.exif == nil {
				return f1.path < f2.path
			}

			if f1.exif == nil {
				return false
			}

			if f2.exif == nil {
				return true
			}

			return !f1.exif.Time.After(*f2.exif.Time)
		},
	)

	_, _ = w.WriteString("<div class=\"gallery\" id=\"")
	_, _ = w.WriteString(ElementID(node.count))
	_, _ = w.WriteString("\">")

	for _, file := range filesExif {
		t := &time.Time{}
		if file.exif != nil {
			t = file.exif.Time
		}
		resPath, ok := g.options.ProvideSource(node.count, file.path, t)
		if !ok {
			continue
		}

		_, _ = w.WriteString("<div class=\"gallery-entry\"><a href=\"")
		_, _ = w.WriteString(resPath)
		_, _ = w.WriteString("\"><img class=\"gallery-item\" src=\"")
		_, _ = w.WriteString(resPath)
		_, _ = w.WriteString("\"")

		if file.exif != nil && file.exif.Time != nil {
			_, _ = w.WriteString(" title=\"")
			_, _ = w.WriteString(file.exif.Time.Format("2006-01-02 15:04:05"))
			_, _ = w.WriteString("\"")
		}

		_, _ = w.WriteString("></a></div>")
	}

	_, _ = w.WriteString("</div>")

	return ast.WalkSkipChildren, nil
}

func addDefaultValues(g *galleryNode) {
	// If include isn't set, assume all JPGs.
	g.Include = strings.TrimSpace(g.Include)
	if len(g.Include) == 0 {
		g.Include = "*.jpg"
	}

	// If no path is set, assume 'photos'
	if len(g.Path) == 0 {
		g.Path = "photos"
	}
}
