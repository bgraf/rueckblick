package gallery

import (
	"fmt"
	"os"
	"path/filepath"
)

type galleryNode struct {
	documentPath string
	count        int
	Path         string `yaml:"path"`
	Include      string `yaml:"include"`
}

func (g *galleryNode) findImagePaths() ([]string, error) {
	docDir := filepath.Dir(g.documentPath)

	pat := filepath.Join(docDir, g.Path, g.Include)

	candidates, err := filepath.Glob(pat)
	if err != nil {
		return nil, fmt.Errorf("glob failed: %w", err)
	}

	writePos := 0
	for _, candidate := range candidates {
		if fs, err := os.Stat(candidate); err != nil || !fs.Mode().IsRegular() {
			continue
		}

		candidates[writePos] = candidate
		writePos++
	}

	candidates = candidates[:writePos]

	return candidates, nil
}
