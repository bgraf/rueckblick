package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GatherFiles(roots []string, extensions []string) ([]string, error) {
	normalizeExtension := func(ext string) (string, bool) {
		ext = strings.ToLower(ext)
		for _, e := range extensions {
			if e == ext {
				return ext, true
			}
		}
		return ext, false
	}

	appendAbsPath := func(paths []string, path string) ([]string, error) {
		path, err := filepath.Abs(path)
		if err != nil {
			return paths, fmt.Errorf("absolute path: %w", err)
		}
		return append(paths, path), nil
	}

	var paths []string

	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			return nil, err
		}

		if fi.Mode().IsRegular() {
			_, ok := normalizeExtension(filepath.Ext(fi.Name()))
			if !ok {
				continue
			}

			paths, err = appendAbsPath(paths, root)
			if err != nil {
				return nil, err
			}

		} else if fi.Mode().IsDir() {
			files, err := os.ReadDir(root)
			if err != nil {
				return nil, fmt.Errorf("read dir: %w", err)
			}

			for _, fi := range files {
				realPath := filepath.Join(root, fi.Name())
				_, ok := normalizeExtension(filepath.Ext(fi.Name()))
				if !ok {
					continue
				}

				paths, err = appendAbsPath(paths, realPath)
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, fmt.Errorf("path '%s' neither directory nor file", root)
		}
	}

	return paths, nil
}
