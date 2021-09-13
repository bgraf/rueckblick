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
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gitlab.com/begraf/rueckblick/markdown/gallery"
	"gitlab.com/begraf/rueckblick/markdown/gpx"
	"gitlab.com/begraf/rueckblick/markdown/yamlblock"
	"gitlab.com/begraf/rueckblick/util/slices"
)

type Store struct {
	RootDirectory       string
	Documents           []*Document
	tagByNormalizedName map[string]Tag
	tags                []Tag
	mediaRewriter       DocumentMediaRewriter
}

func NewStore(rootDirectory string, rewriter DocumentMediaRewriter) (*Store, error) {
	store := &Store{
		RootDirectory:       rootDirectory,
		tagByNormalizedName: make(map[string]Tag),
		mediaRewriter:       rewriter,
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

	rewriter := s.mediaRewriter.MakeRewriter(doc)

	gmark := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			yamlblock.New(
				gallery.NewGalleryAddin(),
				gpx.NewGPXAddin(rewriter),
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

	// Extract YAML meta data
	err = populateFromYAMLMetaData(doc, meta.Get(pc))
	if err != nil {
		return nil, fmt.Errorf("could not parse YAML meta data: %w", err)
	}

	// Add gallery IDs
	galleryCount := gallery.Count(pc)
	for i := 0; i < galleryCount; i++ {
		doc.Galleries = append(doc.Galleries, gallery.ElementID(i))
	}

	return doc, nil
}

func populateFromYAMLMetaData(doc *Document, m map[string]interface{}) error {
	if tags, ok := m["tags"].([]interface{}); ok {
		rawTags, remaining := slices.PartitionStrings(tags)
		for _, rawTag := range rawTags {
			doc.Tags = append(doc.Tags, Tag{Raw: rawTag})
		}

		for _, r := range remaining {
			m, ok := r.(map[interface{}]interface{})
			if !ok {
				continue
			}

			for k, v := range m {
				category, ok := k.(string)
				if !ok {
					continue
				}

				rawItems, ok := v.([]interface{})
				if !ok {
					continue
				}

				rawTags, _ = slices.PartitionStrings(rawItems)
				for _, rawTag := range rawTags {
					doc.Tags = append(doc.Tags, Tag{
						Raw:      rawTag,
						Category: category,
					})
				}
			}
		}
	}

	if title, ok := m["title"].(string); ok {
		doc.Title = title
	}

	var err error

	guidProvided := false
	if guidStr, ok := m["guid"].(string); ok {
		doc.GUID, err = uuid.Parse(guidStr)
		if err == nil {
			guidProvided = true
		}
	}

	if !guidProvided {
		doc.GUID = uuid.New()
	}

	// TODO: allow for more dates, and date ranges etc..
	if dateStr, ok := m["date"].(string); ok {
		doc.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("could not parse date: %w", err)
		}
	} else {
		return fmt.Errorf("field 'date' missing in YAML meta data")
	}

	if abstract, ok := m["abstract"].(string); ok {
		doc.Abstract = strings.TrimSpace(abstract)
	}

	if preview, ok := m["preview"].(string); ok {
		doc.Preview = preview
	}

	return nil
}
