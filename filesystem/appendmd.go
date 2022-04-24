package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func FindAndAppendToMarkdown(directory string, fun func(w io.Writer, path string) error) error {
	files, err := filepath.Glob(filepath.Join(directory, "*.md"))
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	if len(files) != 1 {
		return fmt.Errorf("zero or multiple markdown files in current working directory")
	}

	file := files[0]

	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open document: %w", err)
	}

	defer func() { _ = f.Close() }()

	return fun(f, file)
}
