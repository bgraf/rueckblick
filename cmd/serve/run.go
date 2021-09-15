package serve

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
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

	rootDirectory := config.JournalDirectory()

	rewriter := newRewriter()
	store, err := document.NewStore(
		rootDirectory,
		newDocumentRewriter(rewriter),
	)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.New()

	api := newServeAPI(store, rewriter)
	r.GET("/", api.ServeIndex)
	r.GET("/entry/:GUID", api.ServeEntry)
	r.GET("/resource/:name", api.ServeResource)
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
			if tag.HasCategory() {
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

	r.LoadHTMLGlob("./res/templates/*")
	r.Static("/static", "./res/static")

	if err = r.Run(":8000"); err != nil {
		log.Fatal(err)
	}

	return nil
}

type serveAPI struct {
	store       *document.Store
	tagSet      *render.TagSet
	pathRecoder *render.PathRecoder
	live        bool
	rewriter    *rewriter
}

func newServeAPI(store *document.Store, rewriter *rewriter) *serveAPI {
	store.OrderDocumentsByDate()
	store.OrderTags()

	tagSet := render.NewTagSet()
	pathRecoder := render.NewPathRecoder()

	api := &serveAPI{
		store:       store,
		tagSet:      tagSet,
		pathRecoder: pathRecoder,
		live:        true,
		rewriter:    rewriter,
	}

	for _, doc := range store.Documents {
		api.prepareDocument(doc)
	}

	return api
}

func (api *serveAPI) prepareDocument(doc *document.Document) {
	render.ImplicitFigure(doc)

	api.pathRecoder.RecodeDocument(doc, "/resource")
	for _, t := range doc.Tags {
		api.tagSet.HexColor(t.String())
	}

	if doc.HasPreview() {
		doc.Preview = api.pathRecoder.RecodeDocumentPath(doc, doc.Preview, "/resource")
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

func (api *serveAPI) ServeResource(c *gin.Context) {
	resourceName := c.Param("name")
	resourcePath, ok := api.pathRecoder.Decode(resourceName)
	if !ok {
		c.String(http.StatusNotFound, "not found")
		return
	}

	c.Status(http.StatusOK)
	c.File(resourcePath)
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

	calendarDays := []calendarDay{}

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

	c.HTML(
		http.StatusOK,
		"calendar.html",
		gin.H{
			"Month":     currMonth,
			"NextMonth": nextMonth,
			"PrevMonth": prevMonth,
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
