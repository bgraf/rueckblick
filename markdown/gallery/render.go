package gallery

import (
	"fmt"
	"sort"
	"time"

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
		_ = err // Note: check `err` to see whether EXIF reading succeeded
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
		_, _ = w.WriteString("/file.jpg\"><img class=\"gallery-item\" src=\"")
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
	_, _ = w.WriteString(fmt.Sprintf(
		`<script>
			var lightbox = GLightbox({
				selector: '#%s a',
			});
		</script>`,
		ElementID(node.count),
	))

	return ast.WalkSkipChildren, nil
}
