package render

import (
	"log"
	"net/url"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/data/document"
)

type RecoderFunc func(original string) (string, bool)

func RecodePaths(doc *document.Document, toResource MapToResourceFunc) {
	doc.HTML.Find("img,a,source").Each(func(i int, s *goquery.Selection) {
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

		target, ok := toResource(src)
		if ok {
			s.SetAttr(attribute, target.URI)
		}
	})

}
