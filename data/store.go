package data

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/render"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type StoreOptions struct {
	RenderImagePath func(doc *document.Document, srcPath string) (document.Resource, bool)
}

type Store struct {
	RootDirectory       string
	Documents           []*document.Document
	tagByNormalizedName map[string]document.Tag
	tags                []document.Tag
	options             *StoreOptions
}

func NewStore(rootDirectory string, options *StoreOptions) (*Store, error) {
	store := &Store{
		RootDirectory:       rootDirectory,
		tagByNormalizedName: make(map[string]document.Tag),
		options:             options,
	}

	var err error
	store.Documents, err = store.LoadDocuments(rootDirectory)
	if err != nil {
		return nil, fmt.Errorf("load documents failed: %w", err)
	}

	for _, doc := range store.Documents {
		for _, tag := range doc.Tags {
			name := tag.Normalize()
			if _, ok := store.tagByNormalizedName[name]; !ok {
				store.tagByNormalizedName[name] = tag
				store.tags = append(store.tags, tag)
			}
		}
	}

	return store, nil
}

func (s *Store) OrderDocumentsByDate() {
	sort.Slice(s.Documents, func(i, j int) bool {
		return s.Documents[i].Date.After(s.Documents[j].Date)
	})
}

func (s *Store) DocumentByGUID(guid uuid.UUID) *document.Document {
	for _, doc := range s.Documents {
		if doc.GUID == guid {
			return doc
		}
	}

	return nil
}

func (s *Store) DocumentsOnDate(t time.Time) []*document.Document {
	var docs []*document.Document

	for _, doc := range s.Documents {
		if dates.EqualDate(doc.Date, t) {
			docs = append(docs, doc)
		}
	}

	return docs
}

func (s *Store) ReloadByGUID(guid uuid.UUID) (*document.Document, error) {
	doc := s.DocumentByGUID(guid)
	if doc == nil {
		return nil, fmt.Errorf("no such document")
	}

	newDoc, err := s.LoadDocument(doc.Path)
	if err != nil {
		return nil, fmt.Errorf("new document failed: %w", err)
	}

	newDoc.GUID = doc.GUID
	for i, doc := range s.Documents {
		if newDoc.GUID == doc.GUID {
			s.Documents[i] = newDoc
		}
	}

	return newDoc, nil
}

func (s *Store) DocumentsByTagName(name string) []*document.Document {
	name = document.NormalizeTagName(name)

	var result []*document.Document

	for _, doc := range s.Documents {
		for _, t := range doc.Tags {
			if t.Normalize() == name {
				result = append(result, doc)

				break
			}
		}
	}

	return result
}

func (s *Store) TagByName(name string) (document.Tag, bool) {
	if tag, ok := s.tagByNormalizedName[document.NormalizeTagName(name)]; ok {
		return tag, true
	}

	return document.Tag{}, false
}

func (s *Store) Tags() []document.Tag {
	return s.tags
}

func (s *Store) OrderTags() {
	sort.Slice(
		s.tags,
		func(i, j int) bool {
			return s.tags[i].Normalize() < s.tags[j].Normalize()
		},
	)
}

func (s *Store) LoadDocuments(rootDirectory string) ([]*document.Document, error) {
	var docs []*document.Document

	err := filepath.WalkDir(rootDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.Type().IsRegular() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" {
			return nil
		}

		doc, err := s.LoadDocument(path)
		if err != nil {
			return err
		}

		docs = append(docs, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not load documents: %w", err)
	}

	return docs, nil
}

func (s *Store) LoadDocument(path string) (*document.Document, error) {
	sourceText, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read source file: %w", err)
	}

	doc := &document.Document{
		Path: path,
	}

	sourceText, err = document.ReadFrontMatter(doc, sourceText)
	if err != nil {
		return nil, fmt.Errorf("could not read front matter: %w", err)
	}

	gmark := goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))

	var buffer bytes.Buffer

	pc := parser.NewContext()

	err = gmark.Convert(sourceText, &buffer, parser.WithContext(pc))
	if err != nil {
		log.Fatalf("gmark.Convert: %s", err)
	}

	doc.HTML, err = goquery.NewDocumentFromReader(&buffer)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML: %w", err)
	}

	postprocessDocument(doc, s.options)

	return doc, nil
}

// postprocessDocument modifies the rendered document by replacing links, image and video sources.
func postprocessDocument(doc *document.Document, opts *StoreOptions) {
	render.ImplicitFigure(doc)

	toResource := func(original string) (document.Resource, bool) {
		srcPath := original
		if !filepath.IsAbs(original) {
			srcPath = filepath.Join(filepath.Dir(doc.Path), original)
		}
		resource, _ := opts.RenderImagePath(doc, srcPath)

		return resource, true
	}

	render.RecodePaths(doc, toResource)

	// Must be executed in this order, because GPX requires populated galleries.
	render.EmplaceGalleries(doc, toResource)
	render.EmplaceGPXMaps(doc, toResource)

	if doc.HasPreview() {
		previewResource, _ := toResource(doc.Preview)
		doc.Preview = previewResource.URI
	}
}
