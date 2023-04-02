package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"

	"github.com/PuerkitoBio/goquery"
	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data/document"
	"github.com/bgraf/rueckblick/data/geotrack"
)

// Name of a markdown document tag for GPX tracks
const GPXTagName = "rb-gpx"

// Name of the attribute to specify the track file
const GPXTagTrackAtteName = "track"

// EmplaceGPXMaps replaces all `<rb-gpx ... />` nodes by a collection of nodes representing
// an actual Leaflet map.
//
// Note: requires that the `doc.Galleries` are populated, otherwise matching of images to
// locations will yield no results.
func EmplaceGPXMaps(doc *document.Document, toResource MapToResourceFunc) {
	mapID := -1
	doc.HTML.Find(GPXTagName).Each(func(i int, s *goquery.Selection) {
		mapID++

		trackFile := s.AttrOr(GPXTagTrackAtteName, config.DefaultGPXFile())
		if !path.IsAbs(trackFile) {
			trackFile = path.Join(doc.DocumentDirectory(), trackFile)
		}

		points, images, err := geotrack.LoadTrack(doc, trackFile)
		if err != nil {
			panic(err)
		}

		// Build json payload
		payload := map[string]interface{}{
			"track":  points,
			"images": images,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			panic(err)
		}

		mapElementID := fmt.Sprintf("map-%d", mapID)
		doc.Maps = append(doc.Maps, document.GXPMap{
			GPXPath:   trackFile,
			ElementID: mapElementID,
		})

		payloadStr := string(payloadBytes)

		var buf bytes.Buffer

		_, _ = buf.WriteString(fmt.Sprintf(`<div class="gpx-map" id="%s">`, mapElementID))

		_, _ = buf.WriteString(fmt.Sprintf(`
		<script>
		(function () {
			const mapData = %s;
			let mapContainer = document.currentScript.parentElement;
			window.addEventListener('DOMContentLoaded', function() {
				mountMap(mapContainer, mapData);
			});
		})();
		</script>`,
			payloadStr,
		))
		_, _ = buf.WriteString("</div>")

		s.ReplaceWithHtml(buf.String())
	})
}
