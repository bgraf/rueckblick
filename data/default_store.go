package data

import (
	"fmt"

	"github.com/bgraf/rueckblick/data/document"
)

func NewDefaultStore(journalDirectory string) (*Store, error) {
	storeOpts := &StoreOptions{
		RenderImagePath: func(doc *document.Document, srcPath string) (document.Resource, bool) {
			res := document.Resource{
				URI: fmt.Sprintf("file://%s", srcPath),
			}
			return res, true
		},
	}

	store, err := NewStore(
		journalDirectory,
		storeOpts,
	)

	if err != nil {
		return nil, err
	}

	store.SortDocumentsByDate()
	store.SortTags()

	return store, nil
}
