package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func Abs(p string) string {
	p, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}

	return p
}

func CreateDirectoryIfNotExists(path string) error {
	return os.MkdirAll(path, 0777)
}

func IsDirectory(path string) bool {
	fs, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fs.IsDir()
}

func FileModifiedTime(path string) (mod time.Time, err error) {
	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	mod = fi.ModTime()

	return
}

func FullSubtreeModifiedDate(path string) (time.Time, error) {
	if !IsDirectory(path) {
		return time.Time{}, fmt.Errorf("path '%s' is not a directory", path)
	}

	return fullSubtreeModifiedDate(path)
}

func maxTime(lhs, rhs time.Time) time.Time {
	if lhs.Before(rhs) {
		return rhs
	}

	return lhs
}

func fullSubtreeModifiedDate(path string) (time.Time, error) {
	latest := time.Time{}

	entries, err := os.ReadDir(path)
	if err != nil {
		return latest, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// TODO: handle ret
			subdirMod, err := fullSubtreeModifiedDate(filepath.Join(path, entry.Name()))
			if err != nil {
				return latest, err
			}

			latest = maxTime(latest, subdirMod)
		} else {
			// Handle file
			fi, err := entry.Info()
			if err != nil {
				return latest, fmt.Errorf("query file info: %w", err)
			}

			latest = maxTime(latest, fi.ModTime())
		}
	}

	return latest, nil
}
