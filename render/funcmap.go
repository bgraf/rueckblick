package render

import (
	"fmt"
	"html/template"
	"net/url"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/goodsign/monday"
	"github.com/lucasb-eyer/go-colorful"
)

func MakeTemplateFuncmap() template.FuncMap {
	tagSet := NewTagSet()
	periodColors := make(map[string]colorful.Color)

	return template.FuncMap{
		"tagColor": func(tag document.Tag) string {
			return tagSet.HexColor(tag.String())
		},
		"periodColor": func(name string) string {
			name = document.NormalizeTagName(name)
			if c, ok := periodColors[name]; ok {
				return c.Hex()
			}

			h, _, _ := colorful.WarmColor().Hsv()
			c := colorful.Hsv(h, 0.15, 1.0)
			periodColors[name] = c

			return c.Hex()
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
