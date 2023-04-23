package building

import "github.com/bgraf/rueckblick/data/document"

type DocumentSet struct {
	byPath map[string]*document.Document
}

func NewDocumentSet() *DocumentSet {
	return &DocumentSet{
		byPath: make(map[string]*document.Document),
	}
}

func (s *DocumentSet) Add(d *document.Document) {
	s.byPath[d.Path] = d
}

func (s *DocumentSet) ForEach(f func(*document.Document) error) error {
	for _, d := range s.byPath {
		if err := f(d); err != nil {
			return err
		}

	}
	return nil
}

func (s *DocumentSet) Len() int {
	return len(s.byPath)
}
