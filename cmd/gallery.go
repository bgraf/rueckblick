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
	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/render"
	"github.com/spf13/cobra"
)

// galleryCmd represents the gallery command
var galleryCmd = &cobra.Command{
	Use:   "gallery",
	Short: "Copy a given set of pictures into a common destination folder with optinal scaling",
	Long: `Takes paths of image files or folders which are then searched 
for image files. All found images are copied to a given destination
folder. Optionally, images are scaled.`,

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

	galleryCmd.Flags().IntP("size", "s", 0, "Maximum width or height of the scaled images. If set implies scaling.")
	galleryCmd.Flags().StringP("output", "o", config.DefaultPhotosDirectory(), "Output directory")
}

type genGalleryOptions struct {
	Size                   int
	TargetGalleryDirectory string
	Args                   []string
	DocumentDirectory      string
}

func (opts genGalleryOptions) ShouldScale() bool {
	return opts.Size > 0
}

func defaultGenGalleryOptions() genGalleryOptions {
	return genGalleryOptions{
		Size:                   0,
		TargetGalleryDirectory: config.DefaultPhotosDirectory(),
	}
}

func runGenGallery(cmd *cobra.Command, args []string) error {
	var err error

	opts := defaultGenGalleryOptions()

	if cmd.Flags().Changed("size") {
		opts.Size, err = cmd.Flags().GetInt("size")
		if err != nil {
			log.Fatal(err) // Should not happen
		}

		log.Printf("scaling images to max %d\n", opts.Size)
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
	filePaths, err := filesystem.GatherFiles(opts.Args, []string{".jpeg", ".jpg", ".png"})
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

	// Create thumbs directory
	thumbDirectory := filepath.Join(opts.TargetGalleryDirectory, config.DefaultThumbSubdirectory())
	err = os.MkdirAll(thumbDirectory, 0700)
	if err != nil {
		return fmt.Errorf("create thumb directory: %w", err)
	}

	// Transform images
	imageTransformer := func(srcPath, dstPath string) error {
		err := scaleImage(srcPath, dstPath, opts.Size)
		if err != nil {
			return err
		}
		log.Printf("scaled: %s => %s\n", srcPath, dstPath)
		return nil
	}
	if !opts.ShouldScale() {
		imageTransformer = func(srcPath, dstPath string) error {
			err := filesystem.Copy(srcPath, dstPath)
			if err != nil {
				return err
			}
			log.Printf("copied: %s => %s\n", srcPath, dstPath)
			return nil
		}
	}

	var wg sync.WaitGroup
	srcFiles := make(chan string)
	numCPU := runtime.NumCPU()

	for range numCPU {
		wg.Add(1)
		go func(srcFiles <-chan string) {
			defer wg.Done()
			for path := range srcFiles {
				srcExt := filepath.Ext(path)
				pathWithoutExt := strings.TrimSuffix(path, srcExt)
				nameWithoutExt := filepath.Base(pathWithoutExt)
				dstExt := destinationImageExtension(srcExt)
				dstPath := filepath.Join(opts.TargetGalleryDirectory, nameWithoutExt+dstExt)

				err = imageTransformer(path, dstPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}

				// Create thumbnail
				thumbPath := data.ThumbnailPath(dstPath)
				err = scaleImage(dstPath, thumbPath, config.DefaultThumbWidth())
				if err != nil {
					log.Printf("Thumb: failed: %s => %s", dstPath, err)
				} else {
					log.Printf("Thumb: created %s", thumbPath)

				}
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

	log.Println("done")

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
			dirAttr = fmt.Sprintf(`%s="%s"`, render.GalleryTagDirectoryAttrName, galleryRelPath)
		}

		fmt.Fprintf(f, "\n<%s %s></%s>\n", render.GalleryTagName, dirAttr, render.GalleryTagName)

		return nil
	})
}
