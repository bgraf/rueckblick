package serve

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/document"
	"github.com/bgraf/rueckblick/render"
	"github.com/gin-gonic/gin"
	"github.com/goodsign/monday"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func tagIdentifier(tag string) string {
	tag = document.NormalizeTagName(tag)
	return tag
}

func tagIdentifierEscaped(tag string) string {
	return url.PathEscape(tagIdentifier(tag))
}

func RunServeCmd(cmd *cobra.Command, args []string) error {
	if !config.HasJournalDirectory() {
		return fmt.Errorf("no journal directory configured")
	}

	devMode, err := cmd.Flags().GetBool("dev")
	if err != nil {
		panic(err) // should not happen
	}

	rootDirectory := config.JournalDirectory()

	rewriter := newResourceMap()

	storeOpts := &document.StoreOptions{
		MapGPXResource: func(doc *document.Document, srcPath string) (document.Resource, bool) {
			guid := rewriter.IDFromPath(srcPath)
			res := document.Resource{
				GUID: guid,
				URI:  fmt.Sprintf("/gpx/%s", guid.String()),
			}
			return res, true
		},

		MapImageResource: func(doc *document.Document, galleryNo int, srcPath string) (document.Resource, bool) {
			guid := rewriter.IDFromPath(srcPath)
			res := document.Resource{
				GUID: guid,
				URI:  fmt.Sprintf("/image/%s", guid.String()),
			}
			return res, true
		},
	}

	store, err := document.NewStore(
		rootDirectory,
		storeOpts,
	)
	if err != nil {
		log.Fatal(err)
	}

	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	api := newServeAPI(store, rewriter)
	r.GET("/", api.ServeIndex)
	r.GET("/entry/:GUID", api.ServeEntry)
	r.GET("/image/:GUID", api.ServeImage)
	r.GET("/tags/", api.ServeTags)
	r.GET("/tag/:tag", api.ServeTag)
	r.GET("/calendar/:year/:month", api.ServeCalendar)
	r.GET("/play", api.ServePlay)
	r.GET("/gpx/:GUID", api.ServeGPX)

	r.UseRawPath = true

	r.SetFuncMap(template.FuncMap{
		"tagColor": func(tag document.Tag) string {
			return api.tagSet.HexColor(tag.String())
		},
		"tagIdentifier": func(tag document.Tag) string {
			return tagIdentifierEscaped(tag.String())
		},
		"tagDisplay": func(tag document.Tag) template.HTML {
			if tag.Category == "location" {
				return template.HTML(fmt.Sprintf("<i class=\"icon-map-pin-line icon-small\"></i> %s", tag.String()))
			}

			return template.HTML(tag.String())
		},
		"isFirstOfWeek": func(t time.Time) bool {
			return t.Weekday() == time.Sunday
		},
		"ISOWeek": func(t time.Time) int {
			_, w := t.ISOWeek()
			return w
		},

		"yearMonthDisplay": func(t time.Time) string {
			return monday.Format(t, "January 2006", monday.LocaleDeDE)
		},

		"calendarURL": func(t time.Time) string {
			y, m, _ := t.Date()
			return fmt.Sprintf("/calendar/%d/%d", y, m)
		},

		"today": func() time.Time {
			return time.Now()
		},
	})

	// Load templates and static files
	resourceDir, err := cmd.Flags().GetString("resource-dir")
	if err != nil {
		panic(err)
	}

	if resourceDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("exe name lookup: %w", err)
		}
		fmt.Printf("exe: %s\n", exePath)
		resourceDir = filepath.Dir(exePath)
	}

	r.LoadHTMLGlob(filepath.Join(resourceDir, "res/templates/*"))
	r.Static("/static", filepath.Join(resourceDir, "res/static"))

	// Run
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		panic(err)
	}

	connStr := fmt.Sprintf(":%d", port)

	if err = r.Run(connStr); err != nil {
		log.Fatal(err)
	}

	return nil
}

type serveAPI struct {
	store    *document.Store
	tagSet   *render.TagSet
	live     bool
	rewriter *resourceMap
}

func newServeAPI(store *document.Store, rewriter *resourceMap) *serveAPI {
	store.OrderDocumentsByDate()
	store.OrderTags()

	tagSet := render.NewTagSet()

	api := &serveAPI{
		store:    store,
		tagSet:   tagSet,
		live:     true,
		rewriter: rewriter,
	}

	for _, doc := range store.Documents {
		api.prepareDocument(doc)
	}

	return api
}

func (api *serveAPI) prepareDocument(doc *document.Document) {
	render.ImplicitFigure(doc)

	recoderFunc := func(original string) (string, bool) {
		srcPath := filepath.Join(filepath.Dir(doc.Path), original)
		guid := api.rewriter.IDFromPath(srcPath)
		return fmt.Sprintf("/image/%s", guid.String()), true
	}

	render.RecodePaths(doc, recoderFunc)

	for _, t := range doc.Tags {
		api.tagSet.HexColor(t.String())
	}

	if doc.HasPreview() {
		doc.Preview, _ = recoderFunc(doc.Preview)

	}
}

func (api *serveAPI) documentByGUID(guid uuid.UUID) *document.Document {
	if api.live {
		doc, err := api.store.ReloadByGUID(guid)
		if err != nil {
			log.Println(err)
			return nil
		}

		api.prepareDocument(doc)

		return doc
	}

	return api.store.DocumentByGUID(guid)
}

func (api *serveAPI) ServeEntry(c *gin.Context) {
	guid, err := uuid.Parse(c.Param("GUID"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	doc := api.documentByGUID(guid)
	if doc == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	// Extract body fragment
	fragment, err := doc.HTML.Find("body").Html()
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}

	c.HTML(http.StatusOK, "entry.html", gin.H{
		"Document": doc,
		"Fragment": template.HTML(fragment),
	})
}

func (api *serveAPI) ServeImage(c *gin.Context) {
	guidStr := c.Param("GUID")
	guid, err := uuid.Parse(guidStr)
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	path, ok := api.rewriter.PathFromID(guid)
	if !ok {
		c.String(http.StatusNotFound, "not found")
		return
	}

	c.Status(http.StatusOK)
	c.File(path)
}

func (api *serveAPI) ServeCalendar(c *gin.Context) {
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	byDay := make(map[int][]*document.Document)

	for _, doc := range api.store.Documents {
		y := doc.Date.Year()
		m := int(doc.Date.Month())

		if y != year || m != month {
			continue
		}

		d := doc.Date.Day()

		if docs, ok := byDay[d]; ok {
			byDay[d] = append(docs, doc)
		} else {
			byDay[d] = []*document.Document{doc}
		}
	}

	type calendarDay struct {
		Date     time.Time
		Document *document.Document
	}

	var calendarDays []calendarDay

	curr := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	target := curr.AddDate(0, 1, 0)
	curr = curr.AddDate(0, 0, -int(curr.Weekday()))

	for {
		var doc *document.Document
		if int(curr.Month()) == month {
			if docs, ok := byDay[curr.Day()]; ok {
				doc = docs[0]
			}
		}

		calendarDays = append(calendarDays, calendarDay{
			Document: doc,
			Date:     curr,
		})
		curr = curr.AddDate(0, 0, 1)

		if !curr.Before(target) && curr.Weekday() == time.Sunday {
			break
		}
	}

	currMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	prevMonth := currMonth.AddDate(0, -1, 0)
	nextMonth := currMonth.AddDate(0, 1, 0)
	prevYear := currMonth.AddDate(-1, 0, 0)
	nextYear := currMonth.AddDate(1, 0, 0)

	c.HTML(
		http.StatusOK,
		"calendar.html",
		gin.H{
			"Month":     currMonth,
			"NextMonth": nextMonth,
			"PrevMonth": prevMonth,
			"PrevYear":  prevYear,
			"NextYear":  nextYear,
			"Days":      calendarDays,
		},
	)
}

func (api *serveAPI) ServeTags(c *gin.Context) {
	tags := api.store.Tags()

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].String() < tags[j].String()
	})

	c.HTML(
		http.StatusOK,
		"tags.html",
		tags,
	)
}

func (api *serveAPI) ServePlay(c *gin.Context) {
	c.HTML(
		http.StatusOK,
		"play.html",
		gin.H{},
	)
}
