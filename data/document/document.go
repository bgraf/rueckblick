package document

import (
	"path"
	"path/filepath"
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
	FilePath  string
	Resource  Resource
	Timestamp *time.Time
}

type Gallery struct {
	ElementID string
	Images    []Image
}

func (g *Gallery) AppendImage(res Resource, filePath string, timestamp *time.Time) {
	g.Images = append(
		g.Images,
		Image{
			FilePath:  filePath,
			Resource:  res,
			Timestamp: timestamp,
		},
	)
}

type Document struct {
	// File system path
	Path           string
	HTML           *goquery.Document // HTML content
	GUID           uuid.UUID
	Title          string
	Tags           []Tag
	Date           time.Time
	Abstract       string
	Preview        string
	Galleries      []*Gallery
	Maps           []GXPMap
	HasFrontMatter bool
}

func (doc *Document) DocumentDirectory() string {
	return path.Dir(doc.Path)
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

func (doc *Document) FirstLocationTag() string {
	for _, tag := range doc.Tags {
		if tag.Category == "location" {
			return tag.String()
		}
	}

	return ""
}

func (doc *Document) PreviewAbsolutePath() string {
	if filepath.IsAbs(doc.Preview) {
		return doc.Preview
	}

	return filepath.Join(doc.DocumentDirectory(), doc.Preview)
}
