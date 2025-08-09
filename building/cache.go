package building

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bgraf/rueckblick/data"
)

const cacheFileName = "cache.json"

type buildCache struct {
	Documents []cacheDocument `json:"documents"`
}

type cacheDocument struct {
	Title      string     `json:"title"`
	Tags       []data.Tag `json:"tags"`
	Date       jsonDate   `json:"date"`
	Path       string     `json:"path"`
	OutputPath string     `json:"outputPath"`
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

func readBuildCache(buildDirectory string) (cache buildCache, err error) {
	payloadBytes, err := os.ReadFile(filepath.Join(buildDirectory, cacheFileName))
	if err != nil {
		return
	}

	err = json.Unmarshal(payloadBytes, &cache)
	return
}

func makeBuildCache(state *buildState) buildCache {
	store := state.store

	cache := buildCache{}

	for _, doc := range store.Documents {
		cdoc := cacheDocument{
			Title:      doc.Title,
			Tags:       doc.Tags,
			Date:       jsonDate(doc.Date),
			Path:       doc.Path,
			OutputPath: state.filenamer.EntryFile(doc),
		}

		cache.Documents = append(cache.Documents, cdoc)
	}

	return cache
}

func writeBuildCache(state *buildState) error {
	cache := makeBuildCache(state)

	jsonBytes, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(state.BuildDirectory, cacheFileName), jsonBytes, 0o666)
}
