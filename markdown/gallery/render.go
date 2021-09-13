package gallery

import (
	"sort"

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

	w.WriteString("<div class=\"gallery\" id=\"")
	w.WriteString(ElementID(node.count))
	w.WriteString("\">")

	for _, file := range filesExif {
		w.WriteString("<div class=\"gallery-entry\"><a href=\"")
		w.WriteString(file.path)
		w.WriteString("\"><img class=\"gallery-item\" src=\"")
		w.WriteString(file.path)
		w.WriteString("\"")

		if file.exif != nil && file.exif.Time != nil {
			w.WriteString(" title=\"")
			w.WriteString(file.exif.Time.Format("2006-01-02 15:04:05"))
			w.WriteString("\"")
		}

		w.WriteString("></a></div>")
	}

	w.WriteString("</div>")

	return ast.WalkSkipChildren, nil
}
