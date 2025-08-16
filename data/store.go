package data

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type StoreOptions struct {
	RenderImagePath func(doc *Document, srcPath string) (Resource, bool)
}

type Store struct {
	RootDirectory       string
	Documents           []*Document
	Periods             []Period
	tagByNormalizedName map[string]Tag
	tags                []Tag
	Options             *StoreOptions
}

func NewStore(rootDirectory string, options *StoreOptions) (*Store, error) {
	store := &Store{
		RootDirectory:       rootDirectory,
		tagByNormalizedName: make(map[string]Tag),
		Options:             options,
	}

	var err error
	periodsPath := filepath.Join(rootDirectory, "periods.yaml")
	if filesystem.Exists(periodsPath) {
		store.Periods, err = LoadPeriods(filepath.Join(rootDirectory, "periods.yaml"))
		if err != nil {
			return nil, fmt.Errorf("load periods: %w", err)
		}
	}

	store.Documents, err = store.loadDocuments(rootDirectory)
	if err != nil {
		return nil, fmt.Errorf("load documents: %w", err)
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

func (s *Store) SortDocumentsByDate() {
	sort.Slice(s.Documents, func(i, j int) bool {
		return s.Documents[i].Date.After(s.Documents[j].Date)
	})
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

func (s *Store) Tags() []Tag {
	return s.tags
}

func (s *Store) TagsByCategory() map[string][]Tag {
	groups := make(map[string][]Tag)
	for _, tag := range s.Tags() {
		groups[tag.Category] = append(groups[tag.Category], tag)
	}
	return groups
}

func (s *Store) SortTags() {
	sort.Slice(
		s.tags,
		func(i, j int) bool {
			return s.tags[i].Normalize() < s.tags[j].Normalize()
		},
	)
}

var ErrHtmlNotProcessed = errors.New("HTML not processed")

func (s *Store) GetHtmlFragment(doc *Document) (string, error) {
	if !doc.IsHtmlProcessed {
		return "", ErrHtmlNotProcessed
	}

	return doc.HTML.Find("body").Html()
}

func (s *Store) loadDocuments(rootDirectory string) ([]*Document, error) {
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

		doc, err := s.loadDocument(path)
		if err != nil {
			return err
		}

		// Add additional tags
		for _, period := range s.Periods {
			if !period.From.After(doc.Date) && !period.To.Before(doc.Date) {
				doc.Tags = append(doc.Tags, period.Tag)
				doc.Periods = append(doc.Periods, period)

				break
			}
		}

		docs = append(docs, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not load documents: %w", err)
	}

	return docs, nil
}

func (s *Store) loadDocument(path string) (*Document, error) {
	sourceText, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read source file: %w", err)
	}

	doc := &Document{
		Path:            path,
		IsHtmlProcessed: false,
	}

	sourceText, err = ReadFrontMatter(doc, sourceText)
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

	return doc, nil
}

func ThumbnailPath(image string) string {
	return filepath.Join(filepath.Dir(image), config.DefaultThumbSubdirectory(), filepath.Base(image))
}
