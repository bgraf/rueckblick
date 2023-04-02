package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/bgraf/rueckblick/cmd/tools"
	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/render"
	"github.com/bgraf/rueckblick/util/dates"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/bgraf/rueckblick/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// entryCmd represents the entry command
var entryCmd = &cobra.Command{
	Use:   "entry [PHOTO-AND-TRACK-DIRECTORY]",
	Short: "Interactive process to generate a new entry",
	RunE:  runGenEntry,
}

func init() {
	genCmd.AddCommand(entryCmd)
}

var datePattern = regexp.MustCompile(`\d\d\d\d-\d\d-\d\d`)

func runGenEntry(cmd *cobra.Command, args []string) error {
	if !config.HasJournalDirectory() {
		return fmt.Errorf("no journal directory configured")
	}

	if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	inputDirectory := ""

	if len(args) == 1 {
		inputDirectory = strings.TrimSpace(args[0])
		if s, err := os.Stat(inputDirectory); err != nil || !s.IsDir() {
			return fmt.Errorf("path '%s' is not a directory", inputDirectory)
		}
	}

	date := promptDate(inputDirectory)

	// Read title
	title := ""
	{
		prompt := survey.Input{
			Message: "Title",
		}
		err := survey.AskOne(
			&prompt,
			&title,
			survey.WithValidator(survey.Required),
			survey.WithValidator(
				func(ans interface{}) error {
					normTitle := normalizeTitle(ans.(string))
					if len(normTitle) == 0 {
						return fmt.Errorf("empty normalized title, try letters and digits")
					}
					return nil
				},
			),
		)
		exitOnInterrupt(err)
	}

	var (
		locations []string
		tags      []string
	)

	{
		prompt := survey.Input{
			Message: "Location",
		}
		for {
			location := ""
			err := survey.AskOne(&prompt, &location)
			exitOnInterrupt(err)

			location = strings.TrimSpace(location)
			if len(location) > 0 {
				fmt.Println()
				locations = append(locations, location)
				continue
			}

			break
		}
	}

	{
		prompt := survey.Input{
			Message: "Tag",
		}
		for {
			tag := ""
			err := survey.AskOne(&prompt, &tag)
			exitOnInterrupt(err)

			tag = strings.TrimSpace(tag)
			if len(tag) > 0 {
				fmt.Println()
				tags = append(tags, tag)
				continue
			}

			break
		}
	}

	var abstract string
	{
		prompt := survey.Input{
			Message: "Abstract (optional)",
		}

		err := survey.AskOne(&prompt, &abstract)
		exitOnInterrupt(err)

		abstract = strings.TrimSpace(abstract)
	}

	author := os.Getenv("USER")
	{
		prompt := survey.Input{
			Message: "Author",
			Default: author,
		}
		err := survey.AskOne(
			&prompt,
			&author,
		)
		exitOnInterrupt(err)
	}

	dateStr := dates.DateString(date)

	journalDir := config.JournalDirectory()
	normTitle := normalizeTitle(title)
	entryDirName := fmt.Sprintf("%s-%s", dateStr, normTitle)
	entryDir := filepath.Join(journalDir, fmt.Sprint(date.Year()), entryDirName)
	entryFileName := fmt.Sprintf("%s.md", dateStr)
	entryFile := filepath.Join(entryDir, entryFileName)

	log.Printf("entry directory: %s", entryDir)
	log.Printf("entry file: %s", entryFile)

	// Setup front matter

	var tagMap map[string][]string
	nTags := len(tags) + len(locations)
	if nTags > 0 {
		tagMap = make(map[string][]string)
		if len(tags) > 0 {
			tagMap["general"] = tags
		}
		if len(locations) > 0 {
			tagMap["location"] = locations
		}
	}

	frontMatter := document.FrontMatter{
		Title:    title,
		Date:     document.YamlDate(date),
		Author:   author,
		Tags:     tagMap,
		Abstract: abstract,
	}

	// Review front matter
	{
		err := writeFrontMatter(os.Stdout, frontMatter)
		if err != nil {
			log.Fatal(err)
		}

		isConfirmed := true

		prompt := &survey.Confirm{
			Message: "Proceed",
			Default: isConfirmed,
		}

		err = survey.AskOne(prompt, &isConfirmed)
		exitOnInterrupt(err)

		if !isConfirmed {
			os.Exit(0)
		}
	}

	// Create entry directory
	err := os.MkdirAll(entryDir, 0700)
	if err != nil {
		log.Fatal(err)
	}

	// Create markdown file
	f, err := os.Create(entryFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = writeFrontMatter(f, frontMatter)
	if err != nil {
		log.Fatal(err)
	}

	// Finish
	log.Print("created entry")

	// Generate a gallery if requested by user
	if len(inputDirectory) > 0 {
		if err := copyGpxTracks(inputDirectory, entryDir); err != nil {
			log.Printf("Warning: copying GPX tracks failed: %s\n", err)
		}

		log.Printf("generating gallery from photos-dir: %s\n", inputDirectory)
		if err := generateGallery(inputDirectory, entryDir); err != nil {
			log.Printf("Warning: could not generated gallery: %s\n", err)
		}
	}

	fmt.Printf("== Change to entry directory ==\n\ncd %s\n\n", entryDir)

	return nil
}

func copyGpxTracks(inputDirectory string, entryDirectory string) error {
	// Gather all GPX or NMEA tracks. (NMEA tracks may have .txt extensions)
	filePaths, err := filesystem.GatherFiles([]string{inputDirectory}, []string{".gpx", ".txt"})
	if err != nil {
		return fmt.Errorf("scanning files: %w", err)
	} else if len(filePaths) == 0 {
		return nil
	}

	// Copy all track files to the journal directory.
	for _, inPath := range filePaths {
		outPath := path.Join(entryDirectory, path.Base(inPath))

		err := filesystem.Copy(inPath, outPath)
		if err != nil {
			return err
		}
	}

	// ask whether to append it to the document
	shouldContinue := true

	prompt := &survey.Confirm{
		Message: "Append GPX track to document",
		Default: shouldContinue,
	}

	err = survey.AskOne(prompt, &shouldContinue, nil)
	if err != nil {
		return err
	}

	if !shouldContinue {
		return nil
	}

	// Add all tracks to the document.
	for _, inPath := range filePaths {
		name := path.Base(inPath)
		err := filesystem.FindAndAppendToMarkdown(entryDirectory, func(f io.Writer, path string) error {
			fmt.Fprintf(f, "\n<%s track=\"%s\"></%s>\n", render.GPXTagName, name, render.GPXTagName)
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func generateGallery(photosDirectory string, documentDirectory string) error {
	opts := defaultGenGalleryOptions()
	opts.Args = []string{photosDirectory}
	opts.DocumentDirectory = documentDirectory
	opts.TargetGalleryDirectory = filepath.Join(opts.DocumentDirectory, opts.TargetGalleryDirectory)

	if err := genGallery(opts); err != nil {
		return err
	}

	// Generate preview too?
	prompt := &survey.Confirm{
		Message: "Select a preview image",
		Default: true,
	}

	isConfirmed := true
	err := survey.AskOne(prompt, &isConfirmed)
	exitOnInterrupt(err)

	if isConfirmed {
		err = generatePreview(documentDirectory, opts.TargetGalleryDirectory)
		if err != nil {
			return err
		}
	}

	return nil
}

func generatePreview(documentDirectory string, galleryDirectory string) error {
	sourceImage, err := tools.FehSelectImage(galleryDirectory)
	if err != nil {
		return err
	}

	opts := defaultGenPreviewOptions()
	opts.DocumentDirectory = documentDirectory
	opts.SourceImagePath = sourceImage

	return genPreview(opts)
}

func writeFrontMatter(f io.Writer, fm document.FrontMatter) error {
	fmt.Fprintln(f, "---")

	enc := yaml.NewEncoder(f)
	err := enc.Encode(fm)
	if err != nil {
		return err
	}

	fmt.Fprintln(f, "---")

	return nil
}

func exitOnInterrupt(err error) {
	if err == terminal.InterruptErr {
		os.Exit(1)
	}
}

func normalizeTitle(title string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range title {
		if unicode.IsSpace(r) {
			if !lastDash {
				b.WriteString("-")
				lastDash = true
			}
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
		}
	}

	return b.String()
}

func isTodayAnswer(s string) bool {
	return s == "today" || s == "heute"
}

func promptDate(inputDirectory string) time.Time {
	dateStr := time.Now().Format("2006-01-02")

	isGuessedDate := false
	if match := datePattern.FindString(inputDirectory); match != "" {
		dateStr = match
		isGuessedDate = true
	}

	message := "Date"
	if isGuessedDate {
		message = message + " (guessed)"
	}

	prompt := survey.Input{
		Message: message,
		Default: dateStr,
	}
	err := survey.AskOne(
		&prompt,
		&dateStr,
		survey.WithValidator(func(ans interface{}) error {
			s := ans.(string)

			if isTodayAnswer(s) {
				return nil
			}

			_, err := time.Parse("2006-01-02", s)

			return err
		}),
	)
	exitOnInterrupt(err)

	if isTodayAnswer(dateStr) {
		return time.Now()
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Should not happen due to validator
		panic(err)
	}

	return date
}
