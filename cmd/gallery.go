package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// galleryCmd represents the gallery command
var galleryCmd = &cobra.Command{
	Use:   "gallery",
	Short: "Scale a given set of pictures into a common destination folder",
	Long: `Takes paths of image files or folders which are then searched 
for image files. All found images are scaled and copied to a given destination
folder.`,

	Args: func(cmd *cobra.Command, args []string) error {
		hasError := false

		for _, arg := range args {
			fi, err := os.Stat(arg)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "argument %s: does not exist\n", arg)
				} else {
					fmt.Fprintf(os.Stderr, "argument %s: %s\n", arg, err)
				}
				hasError = true
				continue
			}

			if !fi.IsDir() && !fi.Mode().IsRegular() {
				fmt.Fprintf(os.Stderr, "argument %s: neither directory nor file\n", arg)
				hasError = true
				continue
			}
		}

		if hasError {
			return fmt.Errorf("erroneous arguments")
		}

		return nil
	},

	RunE: runGenGallery,
}

func init() {
	genCmd.AddCommand(galleryCmd)

	galleryCmd.Flags().IntP("size", "s", 2000, "Maximum width or height of the scaled images")
	galleryCmd.Flags().StringP("output", "o", "photos", "Output directory")
	// TODO: bind to config file value
}

func runGenGallery(cmd *cobra.Command, args []string) error {
	maxSize, err := cmd.Flags().GetInt("size")
	if err != nil {
		panic(err) // Should not happen
	}

	outputDir, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err) // Should not happen
	}

	// Gather all image files
	filePaths, err := gatherFiles(args, []string{".jpeg", ".jpg"})
	if err != nil {
		return fmt.Errorf("scanning files: %w", err)
	} else if len(filePaths) == 0 {
		return fmt.Errorf("no files")
	}

	// Create output directory
	err = os.MkdirAll(outputDir, 0700)
	if err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Transform images
	var wg sync.WaitGroup
	srcFiles := make(chan string)
	numCPU := runtime.NumCPU()

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func(srcFiles <-chan string) {
			defer wg.Done()
			for path := range srcFiles {
				srcExt := filepath.Ext(path)
				pathWithoutExt := strings.TrimSuffix(path, srcExt)
				nameWithoutExt := filepath.Base(pathWithoutExt)
				dstExt := destinationImageExtension(srcExt)
				dstPath := filepath.Join(outputDir, nameWithoutExt+dstExt)

				err = scaleImage(path, dstPath, maxSize)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}

				fmt.Printf("scaled: %s\n    to: %s\n", path, dstPath)
			}
		}(srcFiles)
	}

	for _, path := range filePaths {
		srcFiles <- path
	}

	close(srcFiles)

	wg.Wait()

	if err := addGalleryToDocument(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: add to document: %s\n", err)
	}

	fmt.Println("done")

	return nil
}

func scaleImage(src string, dst string, maxSize int) error {
	cmd := exec.Command(
		"convert",
		src,
		"-resize",
		fmt.Sprintf("%dx%d", maxSize, maxSize),
		dst,
	)

	return cmd.Run()
}

func destinationImageExtension(ext string) string {
	ext = strings.ToLower(ext)
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	return ext
}

func gatherFiles(roots []string, extensions []string) ([]string, error) {
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
			ext, ok := normalizeExtension(filepath.Ext(fi.Name()))
			if !ok {
				_, _ = fmt.Fprintf(os.Stderr, "ignoring file with extension '%s': %s\n", ext, root)
				continue
			}

			paths, err = appendAbsPath(paths, root)
			if err != nil {
				return nil, err
			}

		} else if fi.Mode().IsDir() {
			files, err := ioutil.ReadDir(root)
			if err != nil {
				return nil, fmt.Errorf("read dir: %w", err)
			}

			for _, fi := range files {
				realPath := filepath.Join(root, fi.Name())
				ext, ok := normalizeExtension(filepath.Ext(fi.Name()))
				if !ok {
					_, _ = fmt.Fprintf(os.Stderr, "ignoring file with extension '%s': %s\n", ext, realPath)
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

func addGalleryToDocument(galleryOutputDirectory string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current working directory: %w", err)
	}

	galleryRelPath := galleryOutputDirectory
	if filepath.IsAbs(galleryOutputDirectory) {
		galleryRelPath, err = filepath.Rel(cwd, galleryOutputDirectory)
		if err != nil {
			return fmt.Errorf("obtain relative path: %w", err)
		}
	}

	// Find possible document file
	files, err := filepath.Glob(filepath.Join(cwd, "*.md"))
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	if len(files) != 1 {
		return fmt.Errorf("zero or multiple markdown files in current working directory")
	}

	file := files[0]

	// Ask whether to append to gallery
	{
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Append gallery to document (%s)", file),
			Default: false,
		}

		var shouldContinue bool
		err := survey.AskOne(prompt, &shouldContinue, nil)
		if err != nil {
			return err
		}

		if !shouldContinue {
			return nil
		}
	}

	// Open document for appending
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open document: %w", err)
	}

	defer func() { _ = f.Close() }()

	// Append to document file
	fmt.Fprintln(f, "\n:: gallery ---")
	if galleryRelPath != "photos" {
		fmt.Fprintf(f, "path: %s\n", galleryRelPath)
	}
	fmt.Fprintln(f, "---")

	return nil
}
