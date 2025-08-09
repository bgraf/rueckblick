package data

import (
	"fmt"
)

func NewDefaultStore(journalDirectory string) (*Store, error) {
	storeOpts := &StoreOptions{
		RenderImagePath: func(doc *Document, srcPath string) (Resource, bool) {
			res := Resource{
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
