package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/filesystem"
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

	galleryCmd.Flags().IntP("size", "s", config.DefaultPhotoWidth(), "Maximum width or height of the scaled images")
	galleryCmd.Flags().StringP("output", "o", config.DefaultPhotosDirectory(), "Output directory")
}

type genGalleryOptions struct {
	Size                   int
	TargetGalleryDirectory string
	Args                   []string
	DocumentDirectory      string
}

func defaultGenGalleryOptions() genGalleryOptions {
	return genGalleryOptions{
		Size:                   config.DefaultPhotoWidth(),
		TargetGalleryDirectory: config.DefaultPhotosDirectory(),
	}
}

func runGenGallery(cmd *cobra.Command, args []string) error {
	var err error

	opts := defaultGenGalleryOptions()

	opts.Size, err = cmd.Flags().GetInt("size")
	if err != nil {
		log.Fatal(err) // Should not happen
	}

	opts.TargetGalleryDirectory, err = cmd.Flags().GetString("output")
	if err != nil {
		log.Fatal(err) // Should not happen
	}

	opts.DocumentDirectory, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	opts.Args = args

	return genGallery(opts)
}

func genGallery(opts genGalleryOptions) error {
	// Gather all image files
	filePaths, err := filesystem.GatherFiles(opts.Args, []string{".jpeg", ".jpg"})
	if err != nil {
		return fmt.Errorf("scanning files: %w", err)
	} else if len(filePaths) == 0 {
		return fmt.Errorf("no files")
	}

	// Create output directory
	err = os.MkdirAll(opts.TargetGalleryDirectory, 0700)
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
				dstPath := filepath.Join(opts.TargetGalleryDirectory, nameWithoutExt+dstExt)

				err = scaleImage(path, dstPath, opts.Size)
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

	// Add to document if the user wants
	if err := addGalleryToDocument(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: add to document: %s\n", err)
	}

	fmt.Println("done")

	return nil
}

func scaleImage(src string, dst string, maxSize int) error {
	cmd := exec.Command(
		"convert",
		src,
		"-auto-orient",
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

func addGalleryToDocument(opts genGalleryOptions) error {
	var err error

	// Make gallery path relative to document directory
	galleryRelPath := opts.TargetGalleryDirectory
	if filepath.IsAbs(opts.TargetGalleryDirectory) {
		galleryRelPath, err = filepath.Rel(opts.DocumentDirectory, opts.TargetGalleryDirectory)
		if err != nil {
			return fmt.Errorf("obtain relative path: %w", err)
		}
	}

	shouldContinue := true

	prompt := &survey.Confirm{
		Message: "Append gallery to document",
		Default: shouldContinue,
	}

	err = survey.AskOne(prompt, &shouldContinue, nil)
	if err != nil {
		return err
	}

	if !shouldContinue {
		return nil
	}

	return filesystem.FindAndAppendToMarkdown(opts.DocumentDirectory, func(f io.Writer, path string) error {
		dirAttr := ""
		if galleryRelPath != config.DefaultPhotosDirectory() {
			dirAttr = fmt.Sprintf(`directory="%s"`, galleryRelPath)
		}

		fmt.Fprintf(f, "\n<rb-gallery %s />\n", dirAttr)

		return nil
	})
}
