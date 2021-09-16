package document

import (
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type GXPMap struct {
	GPXPath   string
	Resource  Resource
	ElementID string
}

type Image struct {
	FilePath string
	Resource Resource
}

type Gallery struct {
	ElementID string
	Images    []Image
}

func (g *Gallery) AppendImage(res Resource, filePath string) {
	g.Images = append(
		g.Images,
		Image{
			FilePath: filePath,
			Resource: res,
		},
	)
}

type Document struct {
	Path           string            // File system path
	HTML           *goquery.Document // HTML content
	GUID           uuid.UUID
	Title          string
	Tags           []Tag
	Date           time.Time
	Abstract       string
	Preview        string
	Galleries      []*Gallery
	Maps           []GXPMap
	HasFrontmatter bool
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
func (doc *Document) GalleryElementID(no int) string {
	if no < len(doc.Galleries) {
		return doc.Galleries[no].ElementID
	}

	return ""
}
