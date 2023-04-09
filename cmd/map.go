package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data"
	"github.com/bgraf/rueckblick/data/geotrack"
	"github.com/bgraf/rueckblick/filesystem"
	"github.com/bgraf/rueckblick/render"
	"github.com/jftuga/geodist"
	"github.com/spf13/cobra"
)

// mapCmd represents the map command
var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate a map containing far-away points representing single entries.",
	Run:   runMapCmd,
}

func init() {
	buildCmd.AddCommand(mapCmd)
}

type markerData struct {
	URI     string
	Preview string
	Title   string
	Year    int
	LatLng  geotrack.GPXPoint
}

func runMapCmd(cmd *cobra.Command, args []string) {
	journalDirectory := filesystem.Abs(config.JournalDirectory())
	store, err := data.NewDefaultStore(journalDirectory)
	if err != nil {
		log.Fatalf("could not load store: %s\n", err)
	}

	cfgHome := config.HomeCoords()
	home := geodist.Coord{Lat: cfgHome.Lat, Lon: cfgHome.Lon}
	mapThreshold := config.MapThreshold()
	fmt.Printf("Using map threshold %.2fkm\n", mapThreshold)

	var points []markerData

	for _, doc := range store.Documents {
		maps := render.GeoMaps(doc)
		if len(maps) == 0 {
			continue
		}

		maxDist := 0.0
		var maxPoint geotrack.GPXPoint

		for _, m := range maps {
			ps, err := geotrack.LoadTrack(m.GPXPath)
			if err != nil {
				log.Fatalf("could not load track %s: %s", m.GPXPath, err)
			}

			for _, p := range ps {
				candidate := geodist.Coord{Lat: p.Lat, Lon: p.Lon}
				_, dkm := geodist.HaversineDistance(home, candidate)
				if dkm > maxDist {
					maxDist = dkm
					maxPoint = p
				}
			}
		}

		if maxDist >= mapThreshold {
			points = append(points, markerData{
				URI:     render.EntryURL(doc),
				Preview: render.PreviewURL(doc),
				LatLng:  maxPoint,
				Title:   doc.Title,
				Year:    doc.Date.Year(),
			})

			fmt.Printf("Selecting '%s' with %fkm\n", doc.Title, maxDist)
		}
	}

	templates, err := render.ReadTemplates()
	if err != nil {
		log.Fatalf("could not read templates: %s\n", err)
	}

	var mapHTML bytes.Buffer

	payloadBytes, err := json.Marshal(points)
	if err != nil {
		log.Fatalf("could not serialize points: %s\n", err)
	}

	_, _ = mapHTML.WriteString(`<div class="gpx-map" style="height:900px;">`)

	_, _ = mapHTML.WriteString(fmt.Sprintf(`
		<script>
		(function () {
			const mapData = %s;
			let mapContainer = document.currentScript.parentElement;
			window.addEventListener('DOMContentLoaded', function() {
				mountGlobalMap(mapContainer, mapData);
			});
		})();
		</script>`,
		string(payloadBytes),
	))
	_, _ = mapHTML.WriteString("</div>")

	var buf bytes.Buffer

	err = templates.ExecuteTemplate(&buf, "play.html", map[string]interface{}{
		"Map": template.HTML(mapHTML.String()),
	})
	if err != nil {
		log.Fatalf("could not execute template: %s", err)
	}

	buildDirectory := filesystem.Abs(config.BuildDirectory())
	mapFile := filepath.Join(buildDirectory, "globmap.html")

	if err := os.WriteFile(mapFile, buf.Bytes(), 0o666); err != nil {
		log.Fatalf("could not write map file: %s\n", err)
	}
}
