package filesystem

import (
	"os"
)

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
