package render

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/res"
)

var fileNameNormalizationPattern = regexp.MustCompile("[^a-z0-9]")

func EntryFileName(doc *document.Document) string {
	title := NormalizeFileName(doc.Title)
	return fmt.Sprintf("%s-%s.html", doc.Date.Format("2006-01-02"), title)
}

func NormalizeFileName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return fileNameNormalizationPattern.ReplaceAllString(s, "_")
}

func CalendarFileName(year, month int) string {
	return fmt.Sprintf("cal-%04d-%02d.html", year, month)
}

func TagFileName(tag document.Tag) string {
	title := NormalizeFileName(tag.Normalize())
	return fmt.Sprintf("tag-%s.html", title)
}

func PreviewURL(doc *document.Document) string {
	return fmt.Sprintf("file://%s", doc.PreviewAbsolutePath())
}

func EntryURL(doc *document.Document) string {
	return fmt.Sprintf("./%s", EntryFileName(doc))
}

func ReadTemplates() (*template.Template, error) {
	funcMap := MakeTemplateFuncmap()

	funcMap["previewURL"] = func(doc *document.Document) template.URL {
		return template.URL(PreviewURL(doc))
	}

	funcMap["entryURL"] = EntryURL

	funcMap["tagURL"] = func(tag document.Tag) template.URL {
		return template.URL(TagFileName(tag))
	}

	funcMap["calendarURL"] = func(t time.Time) template.URL {
		y, m, _ := t.Date()
		return template.URL(CalendarFileName(y, int(m)))
	}

	templates, err := template.New("").Funcs(funcMap).ParseFS(res.Templates, "templates/*")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return templates, nil
}
