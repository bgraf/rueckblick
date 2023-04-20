package building

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
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

	templates, err := render.ReadTemplates()
	if err != nil {
		return err
	}

	store, err := data.NewDefaultStore(journalDirectory)
	if err != nil {
		return err
	}

	state := &buildState{
		journalDirectory: journalDirectory,
		buildDirectory:   buildDirectory,
		templates:        templates,
		isCleanBuild:     isCleanBuild,
		store:            store,
	}

	numUpdated, err := processEntryFiles(state)
	if err != nil {
		return err
	}

	if numUpdated > 0 {
		periodByDate := make(map[time.Time]document.Period)
		for _, period := range store.Periods {
			dates.ForEachDay(period.From, period.To, func(t time.Time) {
				periodByDate[dates.ToLocal(t)] = period
			})
		}

		getPeriod := func(t time.Time) *document.Period {
			if p, ok := periodByDate[t]; ok {
				return &p
			}

			return nil
		}

		if err := writeIndexFile(state, getPeriod); err != nil {
			return err
		}

		if err := writeTagFiles(state); err != nil {
			return err
		}

		if err := writeTagsIndexFile(state); err != nil {
			return err
		}

		if err := writeCalendarFiles(state, getPeriod); err != nil {
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

type buildState struct {
	journalDirectory string
	buildDirectory   string
	templates        *template.Template
	isCleanBuild     bool
	store            *data.Store
}

type isValidDate = func(t time.Time) bool

func writeCalendarFiles(
	state *buildState,
	getPeriod func(t time.Time) *document.Period,
) error {
	store := state.store

	end := dates.FirstDayOfMonth(store.Documents[0].Date).AddDate(0, 0, 1)
	first := dates.FirstDayOfMonth(store.Documents[len(store.Documents)-1].Date)

	isValid := func(first, last time.Time) isValidDate {
		return func(t time.Time) bool {
			return !t.Before(first) && t.Before(last)
		}
	}(first, end)

	for first.Before(end) {
		err := writeCalendarFile(
			state,
			first.Year(),
			int(first.Month()),
			isValid,
			getPeriod,
		)
		if err != nil {
			return err
		}

		first = dates.AddMonths(first, 1)
	}

	return nil
}

func writeCalendarFile(
	state *buildState,
	year, month int,
	isValidDate isValidDate,
	getPeriod func(t time.Time) *document.Period,
) error {
	type calendarDay struct {
		Date     time.Time
		Document *document.Document
		Period   *document.Period
	}

	var calendarDays []calendarDay

	startDate := dates.FromYM(year, month)
	endDate := dates.LastDayOfMonth(startDate)
	startDate = dates.PriorMonday(startDate)
	endDate = dates.NextSunday(endDate)

	dates.ForEachDay(startDate, endDate, func(curr time.Time) {
		var doc *document.Document

		if docs := state.store.DocumentsOnDate(curr); len(docs) > 0 {
			doc = docs[0]
		}

		calendarDays = append(calendarDays, calendarDay{
			Document: doc,
			Date:     curr,
			Period:   getPeriod(curr),
		})
	})

	currMonth := dates.FromYM(year, month)

	prevMonth := dates.AddMonths(currMonth, -1)
	nextMonth := dates.AddMonths(currMonth, 1)
	prevYear := dates.AddYears(currMonth, -1)
	nextYear := dates.AddYears(currMonth, 1)

	var buf bytes.Buffer
	err := state.templates.ExecuteTemplate(&buf, "calendar.html", map[string]interface{}{
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

	fileName := render.CalendarFileName(year, month)

	calendarFilePath := filepath.Join(state.buildDirectory, fileName)
	err = os.WriteFile(calendarFilePath, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write calendar file: %w", err)
	}

	log.Printf("written calendar file '%s'", calendarFilePath)

	return nil
}

func writeIndexFile(
	state *buildState,
	getPeriod func(t time.Time) *document.Period,
) error {
	groups := render.MakeDocumentGroups(state.store.Documents)
	var buf bytes.Buffer
	err := state.templates.ExecuteTemplate(&buf, "index.html", map[string]interface{}{
		"Groups": groups,
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	indexFile := filepath.Join(state.buildDirectory, "index.html")
	err = os.WriteFile(indexFile, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write index file: %w", err)
	}

	return nil
}

func writeTagsIndexFile(state *buildState) error {
	// Prepare tags
	tagsByCategory := state.store.TagsByCategory()
	tags := []struct {
		Category string
		Tags     []document.Tag
	}{
		{
			Category: "Orte",
			Tags:     tagsByCategory["location"],
		},
		{
			Category: "Personen",
			Tags:     tagsByCategory["people"],
		},
		{
			Category: "Andere",
			Tags:     tagsByCategory["general"],
		},
	}

	for k := range tags {
		ts := tags[k].Tags
		sort.Slice(ts, func(i, j int) bool {
			return ts[i].String() < ts[j].String()
		})
	}

	// Prepare periods
	periods := state.store.Periods
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].From.After(periods[j].From)
	})

	var buf bytes.Buffer
	err := state.templates.ExecuteTemplate(&buf, "tags.html", map[string]any{
		"Tags":    tags,
		"Periods": periods,
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	tagFilePath := filepath.Join(state.buildDirectory, "tags.html")
	err = os.WriteFile(tagFilePath, buf.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("could not write tag file: %w", err)
	}

	return nil
}

func writeTagFiles(state *buildState) error {
	store := state.store

	for _, tag := range store.Tags() {
		documents := store.DocumentsByTagName(tag.Raw)
		groups := render.MakeDocumentGroups(documents)

		var buf bytes.Buffer
		err := state.templates.ExecuteTemplate(&buf, "index.html", map[string]interface{}{
			"Groups": groups,
			"Tag":    tag.Raw,
		})
		if err != nil {
			return fmt.Errorf("could not execute template: %w", err)
		}

		fileName := render.TagFileName(tag)

		tagFilePath := filepath.Join(state.buildDirectory, fileName)
		err = os.WriteFile(tagFilePath, buf.Bytes(), 0666)
		if err != nil {
			return fmt.Errorf("could not write tag file: %w", err)
		}

		log.Printf("written tag file '%s'", tagFilePath)
	}

	return nil
}

func processEntryFiles(state *buildState) (int, error) {
	numUpdated := 0

	for i, doc := range state.store.Documents {
		performUpdate := true

		if !state.isCleanBuild {
			documentModTime, err := filesystem.FullSubtreeModifiedDate(doc.DocumentDirectory())
			if err != nil {
				return 0, err
			}

			entryFile := filepath.Join(state.buildDirectory, render.EntryFileName(doc))
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

		var docSucc *document.Document
		if i > 0 {
			docSucc = state.store.Documents[i-1]
		}

		var docPred *document.Document
		if i+1 < len(state.store.Documents) {
			docPred = state.store.Documents[i+1]
		}

		if err := writeEntryFile(state, doc, docPred, docSucc); err != nil {
			return 0, err
		}

		numUpdated++
	}

	return numUpdated, nil
}

func writeEntryFile(
	state *buildState,
	doc *document.Document,
	docPred *document.Document,
	docSucc *document.Document,
) error {
	// Extract body fragment
	fragment, err := state.store.GetHtmlFragment(doc)
	if err != nil {
		// TODO: log
		return fmt.Errorf("failed to build document: %w", err)
	}

	var buf bytes.Buffer

	err = state.templates.ExecuteTemplate(&buf, "entry.html", map[string]interface{}{
		"Document":     doc,
		"DocumentPred": docPred,
		"DocumentSucc": docSucc,
		"Fragment":     template.HTML(fragment),
	})
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}

	entryFile := filepath.Join(state.buildDirectory, render.EntryFileName(doc))

	err = os.WriteFile(entryFile, buf.Bytes(), 0666)
	if err != nil {
		log.Printf("could not write entry file: %s", err)
	}

	log.Printf("rendered entry '%s'", entryFile)

	return nil
}
