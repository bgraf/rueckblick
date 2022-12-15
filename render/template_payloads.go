package render

import (
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/util/dates"
)

type DocumentGroup struct {
	Documents []*document.Document
	Date      time.Time
}

func MakeDocumentGroups(documents []*document.Document) []DocumentGroup {
	if len(documents) == 0 {
		return nil
	}

	groups := []DocumentGroup{
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
				DocumentGroup{
					Date: ym,
				},
			)
			ci++
		}

		groups[ci].Documents = append(groups[ci].Documents, doc)
	}

	return groups
}
