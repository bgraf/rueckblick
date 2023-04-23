package render

import (
	"fmt"
	"html/template"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/res"
)

type Filenamer interface {
	EntryFile(doc *document.Document) string
	CalendarFile(year, month int) string
	TagFile(tag document.Tag) string
}

func EntryURL(f Filenamer, doc *document.Document) string {
	return fmt.Sprintf("./%s", f.EntryFile(doc))
}

func PreviewURL(f Filenamer, doc *document.Document) string {
	return fmt.Sprintf("file://%s", doc.PreviewAbsolutePath())
}

func ReadTemplates(f Filenamer) (*template.Template, error) {
	funcMap := makeTemplateFuncmap()

	funcMap["previewURL"] = func(doc *document.Document) template.URL {
		return template.URL(PreviewURL(f, doc))
	}

	funcMap["entryURL"] = func(doc *document.Document) string {
		return EntryURL(f, doc)
	}

	funcMap["tagURL"] = func(tag document.Tag) template.URL {
		return template.URL(f.TagFile(tag))
	}

	funcMap["calendarURL"] = func(t time.Time) template.URL {
		y, m, _ := t.Date()
		return template.URL(f.CalendarFile(y, int(m)))
	}

	templates, err := template.New("").Funcs(funcMap).ParseFS(res.Templates, "templates/*")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return templates, nil
}
