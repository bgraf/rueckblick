package serve

import (
	"encoding/json"
	"github.com/bgraf/rueckblick/document"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tkrajina/gpxgo/gpx"
	"net/http"
	"time"
)

func (api *serveAPI) ServeGPX(c *gin.Context) {
	key := c.Param("GUID")

	guid, err := uuid.Parse(key)
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	gpxFilePath, ok := api.rewriter.PathFromID(guid)
	if !ok {
		c.String(http.StatusNotFound, "not found")
		return
	}

	gpxData, err := gpx.ParseFile(gpxFilePath)
	if err != nil {
		_ = c.Error(err)
		c.String(http.StatusInternalServerError, "error during GPX reading")
		return
	}

	track := readGPXTrack(gpxData)

	var locatedImages []locatedImage
	if doc := findDocumentByMapGUID(api.store.Documents, guid); doc != nil {
		locatedImages = findMatchingImages(doc, track)
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"track":  track,
			"images": locatedImages,
		},
	)
}

// findDocumentByTrackGUID searches documents for a document that contains a map of the given guid.
// The matching document is returned or nil, if no document could be found.
func findDocumentByMapGUID(documents []*document.Document, guid uuid.UUID) *document.Document {
	for _, doc := range documents {
		for _, m := range doc.Maps {
			if m.Resource.GUID == guid {
				return doc
			}
		}
	}

	return nil
}

type locatedImage struct {
	URI    string
	LatLng point
}

type point struct {
	lat, lon float64
	time     time.Time
}

func (p point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.lat, p.lon})
}

func readGPXTrack(gpxFile *gpx.GPX) []point {
	var points []point

	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, point{p.Latitude, p.Longitude, p.Timestamp})
			}
		}
	}

	return points
}

func findMatchingImages(doc *document.Document, points []point) []locatedImage {
	var locatedImages []locatedImage

	for _, gal := range doc.Galleries {
		for _, img := range gal.Images {
			if img.Timestamp == nil {
				continue
			}
			targetTime := *img.Timestamp
			nearest, duration := findClosestPointInTime(points, targetTime)

			if duration < 120*time.Second {
				locatedImages = append(
					locatedImages,
					locatedImage{
						URI:    img.Resource.URI,
						LatLng: point{lat: nearest.lat, lon: nearest.lon},
					},
				)
			}
		}
	}

	return locatedImages
}

func findClosestPointInTime(points []point, targetTime time.Time) (point, time.Duration) {
	absDuration := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	durBest := time.Duration(1 << 62)
	iBest := 0
	for i := 0; i < len(points); i++ {
		durI := absDuration(targetTime.Sub(points[i].time))
		if durI <= durBest {
			durBest = durI
			iBest = i
		} else {
			return points[i-1], durBest
		}
	}

	return points[iBest], durBest
}
