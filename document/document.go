package document

import (
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type GXPMap struct {
	GPXPath    string
	ResourceID string
	ElementID  string
}

type Document struct {
	Path      string            // File system path
	HTML      *goquery.Document // HTML content
	GUID      uuid.UUID
	Title     string
	Tags      []Tag
	Date      time.Time
	Abstract  string
	Preview   string
	Galleries []string
	Maps      []GXPMap
}

func (doc *Document) HasAbstract() bool {
	return len(doc.Abstract) > 0
}

func (doc *Document) HasPreview() bool {
	return len(doc.Preview) > 0
}

func (doc *Document) HasGallery() bool {
	return len(doc.Galleries) > 0
}

func (doc *Document) HasMap() bool {
	return len(doc.Maps) > 0
}

func (doc *Document) MapElementID(no int) string {
	if no < len(doc.Maps) {
		return doc.Maps[no].ElementID
	}

	return ""
}
