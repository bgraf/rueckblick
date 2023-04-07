package render

import (
	"fmt"
	"html/template"
	"net/url"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/goodsign/monday"
)

func MakeTemplateFuncmap() template.FuncMap {
	tagSet := NewTagSet()

	return template.FuncMap{
		"tagColor": func(tag document.Tag) string {
			return tagSet.HexColor(tag.String())
		},
		"tagDisplay": func(tag document.Tag) template.HTML {
			if tag.Category == "location" {
				return template.HTML(fmt.Sprintf("<i class=\"icon-map-pin-line icon-small\"></i> %s", tag.String()))
			}

			if tag.Category == "people" {
				return template.HTML(fmt.Sprintf("<i class=\"icon-user icon-small\"></i> %s", tag.String()))
			}

			return template.HTML(tag.String())
		},
		"isFirstOfWeek": func(t time.Time) bool {
			return t.Weekday() == time.Monday
		},
		"ISOWeek": func(t time.Time) int {
			_, w := t.ISOWeek()
			return w
		},

		"yearMonthDisplay": func(t time.Time) string {
			return monday.Format(t, "January 2006", monday.LocaleDeDE)
		},

		"shortenLocation": func(s string) string {
			firstN := func(s string, n int) string {
				i := 0
				for j := range s {
					if i == n {
						return s[:j]
					}
					i++
				}
				return s
			}

			maxLen := 13

			if len(s) > maxLen {
				return firstN(s, maxLen-2) + "..."
			}

			return s
		},

		"today":      time.Now,
		"equalMonth": dates.EqualMonth,
	}

}

func TagIdentifier(tag string) string {
	tag = document.NormalizeTagName(tag)
	return tag
}

func TagIdentifierEscaped(tag string) string {
	return url.PathEscape(TagIdentifier(tag))
}
