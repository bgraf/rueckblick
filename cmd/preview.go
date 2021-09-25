package cmd

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bgraf/rueckblick/document"
	"gopkg.in/yaml.v2"

	"github.com/disintegration/imaging"
	"github.com/spf13/cobra"
)

// previewCmd represents the preview command
var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Generate a square preview image",
	Long: `Preview images are square images associated with journal entries
and are displayed on index pages, tag pages etc.`,
	RunE: runPreview,
}

type genPreviewOptions struct {
	SourceImagePath   string
	TargetImagePath   string
	DocumentDirectory string
	Size              int
}

func defaultGenPreviewOptions() genPreviewOptions {
	return genPreviewOptions{
		Size:            600,
		TargetImagePath: "preview.jpg",
	}
}

func runPreview(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("input image required")
	}

	var err error

	opts := defaultGenPreviewOptions()
	opts.SourceImagePath = args[0]

	opts.TargetImagePath = cmd.Flag("output").Value.String()
	opts.Size, err = cmd.Flags().GetInt("size")
	if err != nil {
		log.Fatal(err) // should not happen
	}

	opts.DocumentDirectory, err = os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not determine current working directory\n")
		os.Exit(1)
	}

	return genPreview(opts)
}

func genPreview(opts genPreviewOptions) error {
	// Read image
	f, err := os.Open(opts.SourceImagePath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("image decode failed: %w", err)
	}

	previewImg := imaging.Fill(img, opts.Size, opts.Size, imaging.Center, imaging.Lanczos)

	outImagePath := opts.TargetImagePath
	if !filepath.IsAbs(outImagePath) {
		outImagePath = filepath.Join(opts.DocumentDirectory, opts.TargetImagePath)
	}

	fOut, err := os.Create(outImagePath)
	if err != nil {
		return fmt.Errorf("could not create output image: %w", err)
	}

	defer fOut.Close()

	jpegOpts := jpeg.Options{Quality: 95}
	err = jpeg.Encode(fOut, previewImg, &jpegOpts)
	if err != nil {
		return fmt.Errorf("saving preview image failed: %w", err)
	}

	fmt.Printf("Created %dx%d preview image '%s'\n", opts.Size, opts.Size, outImagePath)

	{
		err = includeInFrontMatter(opts.DocumentDirectory, outImagePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to include into front matter: %s\n", err)
		}
	}

	return nil
}

// Guides the user to add the generated preview to the front matter of a single markdown
// file in the current working directory.
func includeInFrontMatter(cwd string, outputFilePath string) error {
	files, err := filepath.Glob(filepath.Join(cwd, "*.md"))
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	if len(files) != 1 {
		return fmt.Errorf("zero or multiple markdown files in current working directory")
	}

	file := files[0]

	{
		shouldContinue := true

		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Add preview to front matter (%s)", file),
			Default: shouldContinue,
		}

		err := survey.AskOne(prompt, &shouldContinue, nil)
		if err != nil {
			return err
		}

		if !shouldContinue {
			return nil
		}
	}

	fm, rest, err := readFrontMatterAndSource(file)
	if err != nil {
		return err
	}
	rest = bytes.TrimSpace(rest)

	outputFilePath, err = filepath.Abs(outputFilePath)
	if err != nil {
		return err
	}

	previewPath, err := filepath.Rel(cwd, outputFilePath)
	if err != nil {
		return err
	}

	fm.Preview = previewPath

	// Write document to temporary file
	var newFile string
	{
		f, err := ioutil.TempFile(cwd, "tmp-rb.*.md")
		if err != nil {
			return err
		}

		defer func(f *os.File) {
			_ = f.Close()
		}(f)

		newFile = f.Name()

		_, _ = fmt.Fprintln(f, "---")
		enc := yaml.NewEncoder(f)
		err = enc.Encode(fm)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(f, "---\n\n")
		_, _ = f.Write(rest)
		_, _ = fmt.Fprintln(f)
	}

	// Old permissions
	fi, err := os.Stat(file)
	if err != nil {
		return err
	}

	// Move file
	err = os.Rename(newFile, file)
	if err != nil {
		return err
	}

	err = os.Chmod(file, fi.Mode())
	if err != nil {
		return err
	}

	return nil
}

func readFrontMatterAndSource(file string) (document.FrontMatter, []byte, error) {
	var fm document.FrontMatter

	source, err := ioutil.ReadFile(file)
	if err != nil {
		return fm, nil, err
	}

	fmSrc, rest, err := document.SplitFrontMatterSource(source)
	if err != nil {
		return fm, nil, err
	}

	err = yaml.Unmarshal(fmSrc, &fm)
	if err != nil {
		return fm, nil, err
	}

	return fm, rest, nil
}

func init() {
	genCmd.AddCommand(previewCmd)

	previewCmd.Flags().StringP("output", "o", "preview.jpg", "Output filename")
	previewCmd.Flags().IntP("size", "s", 600, "Preview image width, height")
}
