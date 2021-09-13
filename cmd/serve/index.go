package serve

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/bgraf/rueckblick/document"
	"github.com/bgraf/rueckblick/util/dates"
)

func (api *serveAPI) ServeIndex(c *gin.Context) {
	groups := makeDocumentGroups(api.store.Documents)
	c.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"Groups": groups,
		},
	)
}

func (api *serveAPI) ServeTag(c *gin.Context) {
	tagName := c.Param("tag")

	documents := api.store.DocumentsByTagName(tagName)
	groups := makeDocumentGroups(documents)
	tag, _ := api.store.TagByName(tagName)

	c.HTML(
		http.StatusOK,
		"index.html",
		gin.H{
			"Groups": groups,
			"Tag":    tag.Raw,
		},
	)
}

type documentGroup struct {
	Documents []*document.Document
	Date      time.Time
	HasGap    bool
}

func makeDocumentGroups(documents []*document.Document) []documentGroup {
	if len(documents) == 0 {
		return nil
	}

	groups := []documentGroup{
		{
			Date: dates.FirstDayOfMonth(documents[0].Date),
		},
	}

	ci := 0
	for _, doc := range documents {
		ym := dates.FirstDayOfMonth(doc.Date)
		if ym != groups[ci].Date {
			groups = append(
				groups,
				documentGroup{
					Date:   ym,
					HasGap: ym.AddDate(0, 1, 0) != groups[ci].Date,
				},
			)
			ci++
		}

		groups[ci].Documents = append(groups[ci].Documents, doc)
	}

	return groups
}
