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
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/render"
	"github.com/bgraf/rueckblick/res"
	"github.com/bgraf/rueckblick/util/dates"
)

var fileNameNormalizationPattern = regexp.MustCompile("[^a-z0-9]")

func normalizeFileName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return fileNameNormalizationPattern.ReplaceAllString(s, "_")
}

type Filenamer struct {
}

func (f Filenamer) EntryFile(doc *data.Document) string {
	title := normalizeFileName(doc.Title)
	return fmt.Sprintf("%s-%s.html", doc.Date.Format("2006-01-02"), title)
}
func (f Filenamer) CalendarFile(year, month int) string {
	return fmt.Sprintf("cal-%04d-%02d.html", year, month)
}
func (f Filenamer) TagFile(tag data.Tag) string {
	title := normalizeFileName(tag.Normalize())
	return fmt.Sprintf("tag-%s.html", title)
}

type Options struct {
	Clean            bool
	JournalDirectory string
	BuildDirectory   string
}

func Build(opts Options) error {
	if err := filesystem.CreateDirectoryIfNotExists(opts.BuildDirectory); err != nil {
		return fmt.Errorf("could not ensure build directory: %w", err)
	}

	templates, err := render.ReadTemplates(Filenamer{})
	if err != nil {
		return err
	}

	store, err := data.NewDefaultStore(opts.JournalDirectory)
	if err != nil {
		return err
	}

	state := &buildState{
		Options:   opts,
		templates: templates,
		store:     store,
		filenamer: Filenamer{},
	}
	state.Initialize()

	changedDocuments, err := collectPrimaryChangeDocuments(state)
	if err != nil {
		return err
	}

	// Use the build cache to determine additional documents that need rerendering, because
	// of new neighboring documents.
	//
	// Example: assume the current most recent article is X, without any successor. If
	// we add a new most recent article Y, then X need to point to Y, i.e., X needs to be
	// rerendered despite not being modified.

	currentCache, err := readBuildCache(state.BuildDirectory)
	if err != nil {
		log.Printf("could not read cache: %v\n", err)
	}

	nextCache := makeBuildCache(state)

	log.Printf("curr cache has %d entries\n", len(currentCache.Documents))
	log.Printf("next cache has %d entries\n", len(nextCache.Documents))

	addToDocumentSet := func(entry cacheDocument) {
		for _, doc := range state.store.Documents {
			if doc.Path == entry.Path {
				changedDocuments.Add(doc)
				return
			}
		}
	}

	for i, entry := range nextCache.Documents {
		iOld := -1
		for j, e := range currentCache.Documents {
			if e.Path == entry.Path {
				iOld = j
				break
			}
		}

		if iOld < 0 {
			addToDocumentSet(entry)
			continue
		}

		if (i == 0) != (iOld == 0) {
			// Acquired or lost predecessor => rebuild!
			addToDocumentSet(entry)
			continue
		}

		if i > 0 && iOld > 0 {
			// Compare prior documents
			prior := nextCache.Documents[i-1]
			priorOld := currentCache.Documents[iOld-1]
			if prior.OutputPath != priorOld.OutputPath {
				addToDocumentSet(entry)
				continue
			}
		}

		if (i+1 == len(nextCache.Documents)) != (iOld+1 == len(currentCache.Documents)) {
			// Acquired or lost successor => rebuild!
			addToDocumentSet(entry)
			continue
		}

		if i+1 < len(nextCache.Documents) && iOld+1 < len(currentCache.Documents) {
			succ := nextCache.Documents[i+1]
			succOld := currentCache.Documents[iOld+1]
			if succ.OutputPath != succOld.OutputPath {
				addToDocumentSet(entry)
				continue
			}
		}
	}

	if changedDocuments.Len() == 0 {
		fmt.Println(("nothing to do"))
		return nil
	}

	if err := processEntryFiles(state, changedDocuments); err != nil {
		return err
	}

	if changedDocuments.Len() > 0 {
		periodByDate := make(map[time.Time]data.Period)
		for _, period := range store.Periods {
			dates.ForEachDay(period.From, period.To, func(t time.Time) {
				periodByDate[dates.ToLocal(t)] = period
			})
		}

		getPeriod := func(t time.Time) *data.Period {
			if p, ok := periodByDate[t]; ok {
				return &p
			}

			return nil
		}

		if err := writeIndexFile(state); err != nil {
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
		if err := filesystem.InstallEmbedFS(res.Static, filepath.Join(state.BuildDirectory, "res")); err != nil {
			return fmt.Errorf("installation of state files failed: %w", err)
		}
	}

	if err := writeBuildCache(state); err != nil {
		log.Fatalf("write build cache: %s", err)
	}

	log.Println("done")

	return nil
}

type buildState struct {
	Options
	templates   *template.Template
	store       *data.Store
	indexbyPath map[string]int
	filenamer   Filenamer
}

func (state *buildState) Initialize() {
	indexbyPath := make(map[string]int)
	for i, d := range state.store.Documents {
		indexbyPath[d.Path] = i
	}

	state.indexbyPath = indexbyPath
}

func (state *buildState) Index(d *data.Document) int {
	if idx, ok := state.indexbyPath[d.Path]; ok {
		return idx
	}

	panic("no index")
}

// WriteFile writes a file at the given path interpreted relative to the build directory.
func (state *buildState) WriteFile(path string, content []byte) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path")
	}

	p := filepath.Join(state.BuildDirectory, path)

	return os.WriteFile(p, content, 0o666)
}

func collectPrimaryChangeDocuments(state *buildState) (*DocumentSet, error) {
	s := NewDocumentSet()

	for _, doc := range state.store.Documents {
		performUpdate := true

		if !state.Clean {
			documentModTime, err := filesystem.FullSubtreeModifiedDate(doc.DocumentDirectory())
			if err != nil {
				return nil, err
			}

			entryFile := filepath.Join(state.BuildDirectory, state.filenamer.EntryFile(doc))
			resultModTime, err := filesystem.FileModifiedTime(entryFile)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					performUpdate = true
				} else {
					return nil, err
				}
			} else {
				performUpdate = resultModTime.Before(documentModTime)
			}
		}

		if !performUpdate {
			continue
		}

		s.Add(doc)
	}

	return s, nil
}

type isValidDate = func(t time.Time) bool

func writeCalendarFiles(
	state *buildState,
	getPeriod func(t time.Time) *data.Period,
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
	getPeriod func(t time.Time) *data.Period,
) error {
	type calendarDay struct {
		Date     time.Time
		Document *data.Document
		Period   *data.Period
	}

	var calendarDays []calendarDay

	startDate := dates.FromYM(year, month)
	endDate := dates.LastDayOfMonth(startDate)
	startDate = dates.PriorMonday(startDate)
	endDate = dates.NextSunday(endDate)

	dates.ForEachDay(startDate, endDate, func(curr time.Time) {
		var doc *data.Document

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

	fileName := state.filenamer.CalendarFile(year, month)

	err = state.WriteFile(fileName, buf.Bytes())
	if err != nil {
		return fmt.Errorf("could not write calendar file: %w", err)
	}

	log.Printf("written calendar file '%s'", fileName)

	return nil
}

func writeIndexFile(
	state *buildState,
) error {
	groups := render.MakeDocumentGroups(state.store.Documents)

	/* Group by Years */
	latestYear := 0
	m := make(map[int][]render.DocumentGroup)
	for _, group := range groups {
		year := group.Date.Year()
		latestYear = max(latestYear, year)
		m[year] = append(m[year], group)
	}

	type yearsMenu struct {
		Year       int
		LinkTarget string
	}

	filenameByYear := func(year int) string {
		filename := "index.html"
		if year != latestYear {
			filename = fmt.Sprintf("index_%d.html", year)
		}
		return filename
	}

	var yearMenus []yearsMenu
	for year := range m {
		yearMenus = append(yearMenus, yearsMenu{Year: year, LinkTarget: filenameByYear(year)})
	}

	// Sort decreasing by year
	sort.Slice(yearMenus, func(i, j int) bool {
		return yearMenus[i].Year > yearMenus[j].Year
	})

	for year, groups := range m {
		var buf bytes.Buffer
		err := state.templates.ExecuteTemplate(&buf, "index.html", map[string]interface{}{
			"YearMenus": yearMenus,
			"Groups":    groups,
		})
		if err != nil {
			return fmt.Errorf("could not execute template: %w", err)
		}

		filename := filenameByYear(year)

		err = state.WriteFile(filename, buf.Bytes())
		if err != nil {
			return fmt.Errorf("could not write index file: %w", err)
		}

	}

	return nil
}

func writeTagsIndexFile(state *buildState) error {
	// Prepare tags
	tagsByCategory := state.store.TagsByCategory()
	tags := []struct {
		Category string
		Tags     []data.Tag
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

	err = state.WriteFile("tags.html", buf.Bytes())
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

		fileName := state.filenamer.TagFile(tag)

		err = state.WriteFile(fileName, buf.Bytes())
		if err != nil {
			return fmt.Errorf("could not write tag file: %w", err)
		}

		log.Printf("written tag file '%s'", fileName)
	}

	return nil
}

func processEntryFiles(state *buildState, ds *DocumentSet) error {
	var wg sync.WaitGroup
	docs := make(chan *data.Document)

	for range runtime.NumCPU() {
		wg.Add(1)
		go func(docs <-chan *data.Document) {
			defer wg.Done()
			for doc := range docs {
				i := state.Index(doc)

				var docSucc *data.Document
				if i > 0 {
					docSucc = state.store.Documents[i-1]
				}

				var docPred *data.Document
				if i+1 < len(state.store.Documents) {
					docPred = state.store.Documents[i+1]
				}

				if err := writeEntryFile(state, doc, docPred, docSucc); err != nil {
					// TODO: report error
				}
			}
		}(docs)
	}

	_ = ds.ForEach(func(doc *data.Document) error {
		docs <- doc
		return nil
	})
	close(docs)

	wg.Wait()

	return nil
}

func writeEntryFile(
	state *buildState,
	doc *data.Document,
	docPred *data.Document,
	docSucc *data.Document,
) error {
	render.Render(doc, *state.store.Options)

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

	fileName := state.filenamer.EntryFile(doc)

	err = state.WriteFile(fileName, buf.Bytes())
	if err != nil {
		log.Printf("could not write entry file: %s", err)
	}

	log.Printf("rendered entry '%s'", fileName)

	return nil
}
