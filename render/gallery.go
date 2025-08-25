package render

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/images"
)

// Name of a markdown document tag for GPX tracks
const GalleryTagName = "rb-gallery"

// Name of the attribute to specify the photo directory
const GalleryTagDirectoryAttrName = "directory"

// Name of the attribute to specify the include pattern for file names
const GalleryTagIncludeAttrName = "include"

type MapToResourceFunc func(original string) (data.Resource, bool)

// EmplaceGalleries replaces each `<rb-gallery ... />` node with a collection of nodes representing
// an actual gallery in HTML code.
func EmplaceGalleries(doc *data.Document, toResource MapToResourceFunc) {
	galleryID := -1

	doc.HTML.Find(GalleryTagName).Each(func(i int, s *goquery.Selection) {
		galleryID++

		photoDir := s.AttrOr(GalleryTagDirectoryAttrName, config.DefaultPhotosDirectory())
		if !path.IsAbs(photoDir) {
			photoDir = path.Join(doc.DocumentDirectory(), photoDir)
		}

		pat := s.AttrOr(GalleryTagIncludeAttrName, "*.*")

		files, err := collectGalleryImagePaths(photoDir, pat)
		if err != nil {
			log.Printf("error while collecting gallery images: %s", err)
			return
		}

		filesExif := make(
			[]struct {
				path string
				exif images.EXIFData
			},
			len(files),
		)

		for i, filePath := range files {
			filesExif[i].path = filePath

			var err error

			filesExif[i].exif, err = images.ReadEXIFFromFile(filePath)
			if err != nil {
				log.Printf("EXIF failed: %v => %s\n", filePath, err)

			}
			_ = err // Note: check `err` to see whether EXIF reading succeeded
		}

		sort.Slice(
			filesExif,
			func(i, j int) bool {
				f1 := filesExif[i]
				f2 := filesExif[j]

				if f1.exif.Time.IsNone() && f2.exif.Time.IsNone() {
					return f1.path < f2.path
				}

				if f1.exif.Time.IsNone() {
					return false
				}

				if f2.exif.Time.IsNone() {
					return true
				}

				return !f1.exif.Time.Get().After(f2.exif.Time.Get())
			},
		)

		// Create gallery in document
		galleryElementID := fmt.Sprintf("gallery-%d", galleryID)
		gallery := &data.Gallery{
			ElementID: galleryElementID,
		}
		doc.Galleries = append(doc.Galleries, gallery)

		// Render gallery
		var buf bytes.Buffer

		buf.WriteString(fmt.Sprintf(`<div class="gallery" id="%s">`, galleryElementID))

		for _, file := range filesExif {
			resource, ok := toResource(file.path)
			if !ok {
				continue
			}

			// Fetch thumbnail
			thumb := data.ThumbnailPath(file.path)
			if !filesystem.Exists(thumb) {
				thumb = file.path
				log.Printf("Thumb does not exist, using original file: %s", file.path)
			}
			thumbRes, ok := toResource(thumb)
			if !ok {
				continue
			}

			resPath := resource.URI

			gallery.AppendImage(resource, thumbRes, file.path, file.exif.Time, file.exif.LatLon)

			buf.WriteString("<div class=\"gallery-entry\"><a href=\"")
			buf.WriteString(resPath)
			buf.WriteString("\"><img class=\"gallery-item\" src=\"")
			buf.WriteString(thumbRes.URI)
			buf.WriteString("\"")

			if file.exif.Time.IsSome() {
				buf.WriteString(" title=\"")
				buf.WriteString(file.exif.Time.Get().Format("2006-01-02 15:04:05"))
				buf.WriteString("\"")
			}

			_, _ = buf.WriteString("></a></div>")
		}

		_, _ = buf.WriteString("</div>")
		_, _ = buf.WriteString(fmt.Sprintf(
			`<script>
			var lightbox = GLightbox({
				selector: '#gallery-%d a',
			});
		</script>`,
			galleryID,
		))

		s.ReplaceWithHtml(buf.String())
	})
}

func collectGalleryImagePaths(directory string, pattern string) ([]string, error) {
	pat := path.Join(directory, pattern)

	candidates, err := filepath.Glob(pat)
	if err != nil {
		return nil, fmt.Errorf("glob failed: %w", err)
	}

	writePos := 0
	for _, candidate := range candidates {
		if fs, err := os.Stat(candidate); err != nil || !fs.Mode().IsRegular() {
			continue
		}

		candidates[writePos] = candidate
		writePos++
	}

	candidates = candidates[:writePos]

	return candidates, nil
}
