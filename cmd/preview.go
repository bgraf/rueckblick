package cmd

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"

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

var previewImageWidth *int

func runPreview(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("input image required")
	}

	inputFilePath := args[0]

	f, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("image decode failed: %w", err)
	}

	outputFilePath := cmd.Flag("output").Value.String()
	width := *previewImageWidth
	previewImg := imaging.Fill(img, width, width, imaging.Center, imaging.Lanczos)

	fOut, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("could not create output image: %w", err)
	}

	defer fOut.Close()

	opts := jpeg.Options{Quality: 95}
	err = jpeg.Encode(fOut, previewImg, &opts)
	if err != nil {
		return fmt.Errorf("saving preview image failed: %w", err)
	}

	fmt.Printf("Created %dx%d preview image '%s'\n", width, width, outputFilePath)

	return nil
}

func init() {
	genCmd.AddCommand(previewCmd)

	previewCmd.Flags().StringP("output", "o", "preview.jpg", "Output filename")
	previewImageWidth = previewCmd.Flags().IntP("size", "s", 600, "Preview image width, height")
}
