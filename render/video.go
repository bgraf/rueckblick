package render

import (
	"bytes"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/data/document"
)

// Name of a markdown document tag for videos
const VideoTagName = "rb-video"
const VideoSrcAttributeName = "src"

func EmplaceVideos(doc *document.Document, toResource MapToResourceFunc) {
	fmt.Println("EMPLACE VIDEO")
	fmt.Println(doc.HTML.Html())
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

		var buf bytes.Buffer

		_, _ = buf.WriteString("<figure>")
		_, _ = buf.WriteString("<video controls>")
		_, _ = buf.WriteString(fmt.Sprintf("<source src=\"%s\" type=\"video/mp4\">", srcAttr))
		_, _ = buf.WriteString("</video>")
		_, _ = buf.WriteString("<figcaption></figcaption>")
		_, _ = buf.WriteString("</figure>")

		s.ReplaceWithHtml(buf.String())
	})
}
