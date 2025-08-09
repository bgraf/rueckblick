package render

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/data"
	"golang.org/x/net/html"
)

func ImplicitFigure(doc *data.Document) {
	doc.HTML.Find("p").Each(func(i int, s *goquery.Selection) {
		n := s.Nodes[0]

		if n.FirstChild != n.LastChild {
			return
		}

		if n.FirstChild.Type != html.ElementNode {
			return
		}

		if n.FirstChild.Data != "img" {
			return
		}

		src, ok := s.Children().Attr("src")
		if !ok {
			return
		}

		alt, _ := s.Children().Attr("alt")
		s.Children().Remove()

		s.AppendHtml(fmt.Sprintf(`
			<figure>
				<img src="%s">
				<figcaption>%s</figcaption>
			</figure>`,
			src,
			alt,
		))
	})
}
