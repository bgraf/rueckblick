package render

import (
	"fmt"
	"log"
	"net/url"
	"path"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/document"
	"github.com/google/uuid"
)

type PathRecoder struct {
	PathMap map[string]string
}

func NewPathRecoder() *PathRecoder {
	return &PathRecoder{
		PathMap: make(map[string]string),
	}
}

func (pr *PathRecoder) RecodeDocument(doc *document.Document, routePrefix string) {
	doc.HTML.Find("img,a").Each(func(i int, s *goquery.Selection) {
		attribute := "src"
		if s.Is("a") {
			attribute = "href"
		}

		uri, err := url.Parse(s.AttrOr(attribute, ""))
		if err != nil {
			log.Fatal("error parsing uri: ", err)
		}

		if uri.IsAbs() || filepath.IsAbs(uri.Path) {
			// The URI is not relative, thus we do not change it.
			// TODO: exclude "file:///" absolute uris.
			return
		}

		src, ok := s.Attr(attribute) //TODO: double query of attribute
		if !ok {
			return
		}

		target := pr.RecodeDocumentPath(doc, src, routePrefix)
		s.SetAttr(attribute, target)
	})

}

func (pr *PathRecoder) RecodeDocumentPath(doc *document.Document, docPath string, routePrefix string) string {
	fullSrc := docPath
	if !filepath.IsAbs(fullSrc) {
		var err error

		fullSrc, err = filepath.Abs(path.Join(path.Dir(doc.Path), docPath))
		if err != nil {
			log.Fatal(err)
		}
	}

	target := pr.Recode(fullSrc)
	return path.Join(routePrefix, target)
}

func (pr *PathRecoder) Recode(src string) string {
	target, ok := pr.PathMap[src]
	if !ok {
		uuid, err := uuid.NewUUID()
		if err != nil {
			log.Fatal(err)
		}

		ext := path.Ext(src)
		target = fmt.Sprintf("%s%s", uuid.String(), ext)
		pr.PathMap[src] = target
	}

	return target
}

func (pr *PathRecoder) Decode(name string) (path string, ok bool) {
	var n string

	for path, n = range pr.PathMap {
		if n == name {
			ok = true
			return
		}
	}

	return
}
