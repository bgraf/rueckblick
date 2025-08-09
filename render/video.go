package render

import (
	"bytes"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/data"
	"golang.org/x/net/html"
)

// Name of a markdown document tag for videos
const VideoTagName = "rb-video"
const VideoSrcAttributeName = "src"

func EmplaceVideos(doc *data.Document, toResource MapToResourceFunc) {
	doc.HTML.Find(VideoTagName).Each(func(i int, s *goquery.Selection) {
		srcAttr := strings.TrimSpace(s.AttrOr(VideoSrcAttributeName, ""))
		if len(srcAttr) == 0 {
			log.Printf("Cannot emplace video with missing src-attribute\n")
			return
		}

		if !path.IsAbs(srcAttr) {
			srcAttr = path.Join(doc.DocumentDirectory(), srcAttr)
		}

		// TODO: extract text node and use it as caption
		var texts []string
		for node := range s.Nodes[0].ChildNodes() {
			if node.Type == html.TextNode {
				texts = append(texts, strings.TrimSpace(node.Data))
			}
		}

		var buf bytes.Buffer

		_, _ = buf.WriteString("<figure>")
		_, _ = buf.WriteString("<video controls>")
		_, _ = buf.WriteString(fmt.Sprintf("<source src=\"%s\" type=\"video/mp4\">", srcAttr))
		_, _ = buf.WriteString("</video>")
		_, _ = buf.WriteString(fmt.Sprintf("<figcaption>%s</figcaption>", strings.Join(texts, " ")))
		_, _ = buf.WriteString("</figure>")

		s.ReplaceWithHtml(buf.String())
	})
}
