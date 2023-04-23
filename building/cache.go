package building

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bgraf/rueckblick/data/document"
)

const cacheFileName = "cache.json"

type buildCache struct {
	Documents []cacheDocument `json:"documents"`
}

type cacheDocument struct {
	Title string         `json:"title"`
	Tags  []document.Tag `json:"tags"`
	Date  jsonDate       `json:"date"`
	Path  string         `json:"path"`
}

type jsonDate time.Time

func (j jsonDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(j).Format("2006-01-02"))
}

func (j *jsonDate) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	*j = jsonDate(t)
	return nil
}

func writeBuildCache(state *buildState) error {
	store := state.store

	cache := buildCache{}

	for _, doc := range store.Documents {
		cdoc := cacheDocument{
			Title: doc.Title,
			Tags:  doc.Tags,
			Date:  jsonDate(doc.Date),
			Path:  doc.Path,
		}

		cache.Documents = append(cache.Documents, cdoc)
	}

	jsonBytes, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(state.BuildDirectory, cacheFileName), jsonBytes, 0o666)
}
