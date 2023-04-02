package building

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/render"
	"github.com/bgraf/rueckblick/res"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/spf13/cobra"
)

func RunBuildCmd(cmd *cobra.Command, args []string) error {
	if !config.HasJournalDirectory() {
		return fmt.Errorf("no journal directory configured")
	}

	if !config.HasBuildDirectory() {
		return fmt.Errorf("no build directory configured")
	}

	isCleanBuild, err := cmd.Flags().GetBool("clean")
	if err != nil {
		return err
	}

	journalDirectory := filesystem.Abs(config.JournalDirectory())
	buildDirectory := filesystem.Abs(config.BuildDirectory())

	log.Printf("journal directory: %s", journalDirectory)
	log.Printf("build directory:   %s", buildDirectory)

	if err := filesystem.CreateDirectoryIfNotExists(buildDirectory); err != nil {
		return fmt.Errorf("could not ensure build directory: %w", err)
	}

	templates, err := readTemplates()
	if err != nil {
		return err
	}

	store, err := readStore(journalDirectory)
	if err != nil {
		return err
	}

	numUpdated, err := processEntryFiles(store, templates, buildDirectory, isCleanBuild)
	if err != nil {
		return err
	}

	if numUpdated > 0 {
		if err := writeIndexFile(store, templates, buildDirectory); err != nil {
			return err
		}

		if err := writeTagFiles(store, templates, buildDirectory); err != nil {
			return err
		}

		if err := writeTagsIndexFile(store, templates, buildDirectory); err != nil {
			return err
		}

		if err := writeCalendarFiles(store, templates, buildDirectory); err != nil {
			return err
		}

		// TODO: replace constant "res" by some globally configurable value
		if err := filesystem.InstallEmbedFS(res.Static, filepath.Join(buildDirectory, "res")); err != nil {
			return fmt.Errorf("installation of state files failed: %w", err)
		}
	}

	log.Println("done")
	os.Exit(0)

	return nil
}

type isValidDate = func(t time.Time) bool

func writeCalendarFiles(store *data.Store, templates *template.Template, buildDirectory string) error {
	end := dates.FirstDayOfMonth(store.Documents[0].Date).AddDate(0, 0, 1)
	first := dates.FirstDayOfMonth(store.Documents[len(store.Documents)-1].Date)

	isValid := func(first, last time.Time) isValidDate {
		return func(t time.Time) bool {
			return !t.Before(first) && t.Before(last)
		}
	}(first, end)

	for first.Before(end) {
		if err := writeCalendarFile(store, templates, buildDirectory, first.Year(), int(first.Month()), isValid); err != nil {
			return err
		}

		first = dates.AddMonths(first, 1)
	}

	return nil
}

func writeCalendarFile(store *data.Store, templates *template.Template, buildDirectory string, year, month int, isValidDate isValidDate) error {
	type calendarDay struct {
		Date     time.Time
		Document *document.Document
	}

	var calendarDays []calendarDay

	startDate := dates.FromYM(year, month)
	endDate := dates.LastDayOfMonth(startDate)
	startDate = dates.PriorMonday(startDate)
	endDate = dates.NextSunday(endDate)

	dates.ForEachDay(startDate, endDate, func(curr time.Time) {
		var doc *document.Document

		if docs := store.DocumentsOnDate(curr); len(docs) > 0 {
			doc = docs[0]
		}

		calendarDays = append(calendarDays, calendarDay{
			Document: doc,
			Date:     curr,
		})
	})

	currMonth := dates.FromYM(year, month)

	prevMonth := dates.AddMonths(currMonth, -1)
	nextMonth := dates.AddMonths(currMonth, 1)
	prevYear := dates.AddYears(currMonth, -1)
	nextYear := dates.AddYears(currMonth, 1)

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "calendar.html", map[string]interface{}{
		"Month":        currMonth,
		"PrevMonth":    prevMonth,
		"NextMonth":    nextMonth,
		"PrevYear":     prevYear,
		"NextYear":     nextYear,
		"HasPrevMonth": isValidDate(prevMonth),
		"HasNextMonth": isValidDate(nextMonth),
		"HasPrevYear":  isValidDate(prevYear),
		"HasNextYear":  isValidDate(nextYear),
		"Days":         calendarDays,
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	fileName := calendarFileName(year, month)

	calendarFilePath := filepath.Join(buildDirectory, fileName)
	err = os.WriteFile(calendarFilePath, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write calendar file: %w", err)
	}

	log.Printf("written calendar file '%s'", calendarFilePath)

	return nil
}

func writeIndexFile(store *data.Store, templates *template.Template, buildDirectory string) error {
	groups := render.MakeDocumentGroups(store.Documents)
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "index.html", map[string]interface{}{
		"Groups": groups,
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	indexFile := filepath.Join(buildDirectory, "index.html")
	err = os.WriteFile(indexFile, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write index file: %w", err)
	}

	return nil
}

func writeTagsIndexFile(store *data.Store, templates *template.Template, buildDirectory string) error {
	tags := store.Tags()

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].String() < tags[j].String()
	})

	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "tags.html", tags)
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	tagFilePath := filepath.Join(buildDirectory, "tags.html")
	err = os.WriteFile(tagFilePath, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write tag file: %w", err)
	}

	return nil
}

func writeTagFiles(store *data.Store, templates *template.Template, buildDirectory string) error {
	for _, tag := range store.Tags() {
		documents := store.DocumentsByTagName(tag.Raw)
		groups := render.MakeDocumentGroups(documents)

		var buf bytes.Buffer
		err := templates.ExecuteTemplate(&buf, "index.html", map[string]interface{}{
			"Groups": groups,
			"Tag":    tag.Raw,
		})
		if err != nil {
			return fmt.Errorf("could not execute template: %w", err)
		}

		fileName := tagFileName(tag)

		tagFilePath := filepath.Join(buildDirectory, fileName)
		err = os.WriteFile(tagFilePath, buf.Bytes(), 0666)
		if err != nil {
			return fmt.Errorf("could not write tag file: %w", err)
		}

		log.Printf("written tag file '%s'", tagFilePath)
	}

	return nil
}

func processEntryFiles(store *data.Store, templates *template.Template, buildDirectory string, isCleanBuild bool) (int, error) {
	numUpdated := 0

	for _, doc := range store.Documents {
		performUpdate := true

		if !isCleanBuild {
			documentModTime, err := filesystem.FullSubtreeModifiedDate(doc.DocumentDirectory())
			if err != nil {
				return 0, err
			}

			entryFile := filepath.Join(buildDirectory, entryFileName(doc))
			resultModTime, err := filesystem.FileModifiedTime(entryFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					performUpdate = true
				} else {
					return 0, err
				}
			} else {
				performUpdate = resultModTime.Before(documentModTime)
			}
		}

		if !performUpdate {
			continue
		}

		if err := writeEntryFile(store, doc, templates, buildDirectory); err != nil {
			return 0, err
		}

		numUpdated++
	}

	return numUpdated, nil
}

func writeEntryFile(store *data.Store, doc *document.Document, templates *template.Template, buildDirectory string) error {
	// Extract body fragment
	fragment, err := store.GetHtmlFragment(doc)
	if err != nil {
		// TODO: log
		return fmt.Errorf("failed to build document: %w", err)
	}

	var buf bytes.Buffer

	err = templates.ExecuteTemplate(&buf, "entry.html", map[string]interface{}{
		"Document": doc,
		"Fragment": template.HTML(fragment),
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	entryFile := filepath.Join(buildDirectory, entryFileName(doc))

	err = os.WriteFile(entryFile, buf.Bytes(), 0666)
	if err != nil {
		log.Printf("could not write entry file: %s", err)
	}

	log.Printf("rendered entry '%s'", entryFile)

	return nil
}

func calendarFileName(year, month int) string {
	return fmt.Sprintf("cal-%04d-%02d.html", year, month)
}

func tagFileName(tag document.Tag) string {
	title := normalizeFileName(tag.Normalize())
	return fmt.Sprintf("tag-%s.html", title)
}

func entryFileName(doc *document.Document) string {
	title := normalizeFileName(doc.Title)
	return fmt.Sprintf("%s-%s.html", doc.Date.Format("2006-01-02"), title)
}

var fileNameNormalizationPattern = regexp.MustCompile("[^a-z0-9]")

func normalizeFileName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return fileNameNormalizationPattern.ReplaceAllString(s, "_")
}

func readStore(journalDirectory string) (*data.Store, error) {
	storeOpts := &data.StoreOptions{
		RenderImagePath: func(doc *document.Document, srcPath string) (document.Resource, bool) {
			res := document.Resource{
				URI: fmt.Sprintf("file://%s", srcPath),
			}
			return res, true
		},
	}

	store, err := data.NewStore(
		journalDirectory,
		storeOpts,
	)

	if err != nil {
		return nil, err
	}

	store.SortDocumentsByDate()
	store.SortTags()

	return store, nil
}

func readTemplates() (*template.Template, error) {
	funcMap := render.MakeTemplateFuncmap()

	funcMap["previewURL"] = func(doc *document.Document) template.URL {
		return template.URL(fmt.Sprintf("file://%s", doc.PreviewAbsolutePath()))
	}

	funcMap["entryURL"] = func(doc *document.Document) string {
		return fmt.Sprintf("./%s", entryFileName(doc))
	}

	funcMap["tagURL"] = func(tag document.Tag) template.URL {
		return template.URL(tagFileName(tag))
	}

	funcMap["calendarURL"] = func(t time.Time) template.URL {
		y, m, _ := t.Date()
		return template.URL(calendarFileName(y, int(m)))
	}

	// TODO: replace by relative string or embed
	templates, err := template.New("").Funcs(funcMap).ParseFS(res.Templates, "templates/*")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return templates, nil
}
