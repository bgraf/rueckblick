package building

import "github.com/bgraf/rueckblick/data"

type DocumentSet struct {
	byPath map[string]*data.Document
}

func NewDocumentSet() *DocumentSet {
	return &DocumentSet{
		byPath: make(map[string]*data.Document),
	}
}

func (s *DocumentSet) Add(d *data.Document) {
	s.byPath[d.Path] = d
}

func (s *DocumentSet) ForEach(f func(*data.Document) error) error {
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
