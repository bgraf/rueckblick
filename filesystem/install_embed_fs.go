package filesystem

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func InstallEmbedFS(fs embed.FS, root string) error {
	return installEmbedFSDirectory(fs, ".", root)
}

func installEmbedFSDirectory(fs embed.FS, embedDirectory string, targetDirectory string) error {
	if err := CreateDirectoryIfNotExists(targetDirectory); err != nil {
		return fmt.Errorf("creating root directory '%s' failed: %w", targetDirectory, err)
	}

	entries, err := fs.ReadDir(embedDirectory)
	if err != nil {
		return fmt.Errorf("could not read embedded FS: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Descent into subdirectory
			subEmbedDirectory := filepath.Join(embedDirectory, entry.Name())
			subTargetDirectory := filepath.Join(targetDirectory, entry.Name())
			if err = installEmbedFSDirectory(fs, subEmbedDirectory, subTargetDirectory); err != nil {
				return fmt.Errorf("could not install subdirectory: %w", err)
			}
		} else {
			// Install file
			embedFile := filepath.Join(embedDirectory, entry.Name())
			targetFile := filepath.Join(targetDirectory, entry.Name())

			log.Printf("installing '%s'", embedFile)

			content, err := fs.ReadFile(embedFile)
			if err != nil {
				return fmt.Errorf("could not read embedded file '%s': %w", embedFile, err)
			}

			if err := os.WriteFile(targetFile, content, 0666); err != nil {
				return fmt.Errorf("could not write file '%s': %w", targetFile, err)
			}
		}
	}

	return nil
}
