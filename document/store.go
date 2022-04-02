package document

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
	"github.com/bgraf/rueckblick/markdown/gallery"
	"github.com/bgraf/rueckblick/markdown/gpx"
	"github.com/bgraf/rueckblick/markdown/yamlblock"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Store struct {
	RootDirectory       string
	Documents           []*Document
	tagByNormalizedName map[string]Tag
	tags                []Tag
	options             *StoreOptions
}

func NewStore(rootDirectory string, options *StoreOptions) (*Store, error) {
	store := &Store{
		RootDirectory:       rootDirectory,
		tagByNormalizedName: make(map[string]Tag),
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

func (s *Store) DocumentByGUID(guid uuid.UUID) *Document {
	for _, doc := range s.Documents {
		if doc.GUID == guid {
			return doc
		}
	}

	return nil
}

func (s *Store) DocumentsOnDate(t time.Time) []*Document {
	var docs []*Document

	for _, doc := range s.Documents {
		if dates.EqualDate(doc.Date, t) {
			docs = append(docs, doc)
		}
	}

	return docs
}

func (s *Store) ReloadByGUID(guid uuid.UUID) (*Document, error) {
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

func (s *Store) DocumentsByTagName(name string) []*Document {
	name = NormalizeTagName(name)

	var result []*Document

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

func (s *Store) TagByName(name string) (Tag, bool) {
	if tag, ok := s.tagByNormalizedName[NormalizeTagName(name)]; ok {
		return tag, true
	}

	return Tag{}, false
}

func (s *Store) Tags() []Tag {
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

func (s *Store) LoadDocuments(rootDirectory string) ([]*Document, error) {
	var docs []*Document

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

func (s *Store) LoadDocument(path string) (*Document, error) {
	sourceText, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read source file: %w", err)
	}

	doc := &Document{
		Path: path,
	}

	sourceText, err = readFrontMatter(doc, sourceText)
	if err != nil {
		return nil, fmt.Errorf("could not read front matter: %w", err)
	}

	gpxOpts := &gpx.Options{
		ProvideSource: func(mapNo int, srcPath string) (string, bool) {
			res, ok := s.options.MapGPXResource(doc, srcPath)
			if !ok {
				return "", false
			}

			doc.Maps = append(
				doc.Maps,
				GXPMap{
					GPXPath:   srcPath,
					Resource:  res,
					ElementID: gpx.ElementID(mapNo),
				},
			)

			return res.URI, true
		},
	}

	galleryOpts := &gallery.Options{
		ProvideSource: func(galleryNo int, srcPath string, timestamp *time.Time) (string, bool) {
			res, ok := s.options.MapImageResource(doc, galleryNo, srcPath)
			if !ok {
				return "", false
			}

			for len(doc.Galleries) <= galleryNo {
				doc.Galleries = append(
					doc.Galleries,
					&Gallery{ElementID: gallery.ElementID(len(doc.Galleries))},
				)
			}

			doc.Galleries[galleryNo].AppendImage(res, srcPath, timestamp)

			return res.URI, true
		},
	}

	gmark := goldmark.New(
		goldmark.WithExtensions(
			yamlblock.New(
				gallery.NewGalleryAddin(galleryOpts),
				gpx.NewGPXAddin(gpxOpts),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	var buffer bytes.Buffer

	pc := parser.NewContext()
	yamlblock.SetDocumentPath(pc, path)

	err = gmark.Convert(sourceText, &buffer, parser.WithContext(pc))
	if err != nil {
		log.Fatalf("gmark.Convert: %s", err)
	}

	doc.HTML, err = goquery.NewDocumentFromReader(&buffer)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML: %w", err)
	}

	return doc, nil
}
