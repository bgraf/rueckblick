package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/bgraf/rueckblick/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// entryCmd represents the entry command
var entryCmd = &cobra.Command{
	Use:   "entry",
	Short: "Interactive process to generate a new entry",
	RunE:  runGenEntry,
}

func init() {
	genCmd.AddCommand(entryCmd)
}

func runGenEntry(cmd *cobra.Command, args []string) error {
	if !config.HasJournalDirectory() {
		return fmt.Errorf("no journal directory configured")
	}

	// Read date
	dateStr := time.Now().Format("2006-01-02")
	{
		prompt := survey.Confirm{
			Message: fmt.Sprintf("Today (%s)", dateStr),
			Default: true,
		}
		isToday := true
		err := survey.AskOne(&prompt, &isToday)
		exitOnInterrupt(err)

		if !isToday {
			prompt := survey.Input{
				Message: "Date",
			}
			err := survey.AskOne(
				&prompt,
				&dateStr,
				survey.WithValidator(func(ans interface{}) error {
					_, err := time.Parse("2006-01-02", ans.(string))
					return err
				}),
			)
			exitOnInterrupt(err)
		}

	}

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

	tags := []string{}
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

	fmt.Println("date: ", dateStr)
	fmt.Println("title: ", title)
	fmt.Println("author: ", author)
	fmt.Printf("Tags: %v\n", tags)

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Should not happen due to validator
		panic(err)
	}

	journalDir := config.JournalDirectory()
	normTitle := normalizeTitle(title)
	entryDirName := fmt.Sprintf("%s-%s", dateStr, normTitle)
	entryDir := filepath.Join(journalDir, fmt.Sprint(date.Year()), entryDirName)
	entrFileName := fmt.Sprintf("%s.md", dateStr)
	entryFile := filepath.Join(entryDir, entrFileName)

	log.Printf("entry directory: %s", entryDir)
	log.Printf("entry file: %s", entryFile)

	// Create entry directory
	err = os.MkdirAll(entryDir, 0700)
	if err != nil {
		log.Fatal(err)
	}

	// Create markdown file
	f, err := os.Create(entryFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Write front-matter
	fmt.Fprintln(f, "---")

	frontMatter := struct {
		Title   string   `yaml:"title"`
		DateStr string   `yaml:"date"`
		Author  string   `yaml:"author"`
		Tags    []string `yaml:"tags"`
	}{
		Title:   title,
		DateStr: dateStr,
		Author:  author,
		Tags:    tags,
	}

	enc := yaml.NewEncoder(f)
	err = enc.Encode(frontMatter)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(f, "---")

	fmt.Fprintf(f, "\nHello!\n")

	// Finish
	log.Print("created entry")

	fmt.Printf("== Change to entry directory ==\n\ncd %s\n\n", entryDir)

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
