package render

import (
	"time"

	"github.com/bgraf/rueckblick/document"
	"github.com/bgraf/rueckblick/util/dates"
)

type DocumentGroup struct {
	Documents []*document.Document
	Date      time.Time
	HasGap    bool
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
