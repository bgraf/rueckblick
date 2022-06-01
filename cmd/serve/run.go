package serve

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/render"
	"github.com/bgraf/rueckblick/util/dates"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

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

	storeOpts := &data.StoreOptions{
		RenderImagePath: func(doc *document.Document, srcPath string) (document.Resource, bool) {
			guid := rewriter.IDFromPath(srcPath)
			res := document.Resource{
				GUID: guid,
				URI:  fmt.Sprintf("/image/%s", guid.String()),
			}
			return res, true
		},
	}

	store, err := data.NewStore(
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
	r.GET("/image/:GUID/file.jpg", api.ServeImage)
	r.GET("/tags/", api.ServeTags)
	r.GET("/tag/:tag", api.ServeTag)
	r.GET("/calendar/:year/:month", api.ServeCalendar)
	r.GET("/play", api.ServePlay)
	r.GET("/gpx/:GUID", api.ServeGPX)

	r.UseRawPath = true

	r.SetFuncMap(render.MakeTemplateFuncmap())

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
	store    *data.Store
	live     bool
	rewriter *resourceMap
}

func newServeAPI(store *data.Store, rewriter *resourceMap) *serveAPI {
	store.OrderDocumentsByDate()
	store.OrderTags()

	api := &serveAPI{
		store:    store,
		live:     true,
		rewriter: rewriter,
	}

	for _, doc := range store.Documents {
		api.prepareDocument(doc)
	}

	return api
}

func (api *serveAPI) prepareDocument(doc *document.Document) {
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

		if docs := api.store.DocumentsOnDate(curr); len(docs) > 0 {
			doc = docs[0]
		}

		calendarDays = append(calendarDays, calendarDay{
			Document: doc,
			Date:     curr,
		})
	})

	currMonth := dates.FromYM(year, month)

	c.HTML(
		http.StatusOK,
		"calendar.html",
		gin.H{
			"Month":     currMonth,
			"PrevMonth": dates.AddMonths(currMonth, -1),
			"NextMonth": dates.AddMonths(currMonth, 1),
			"PrevYear":  dates.AddYears(currMonth, -1),
			"NextYear":  dates.AddYears(currMonth, 1),
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
