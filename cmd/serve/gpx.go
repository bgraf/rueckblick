package serve

import (
	"encoding/json"
	"fmt"
	"github.com/bgraf/rueckblick/document"
	"github.com/bgraf/rueckblick/images"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tkrajina/gpxgo/gpx"
	"log"
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

	locatedImages := loadMatchingImages(api, guid, gpxData)

	c.JSON(
		http.StatusOK,
		gin.H{
			"track":  track,
			"images": locatedImages,
		},
	)
}

type locatedImage struct {
	URI    string
	LatLng point
}

type point struct {
	lat, lon float64
}

func (p point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.lat, p.lon})
}

func readGPXTrack(gpxFile *gpx.GPX) []point {
	var points []point

	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, point{p.Latitude, p.Longitude})
			}
		}
	}

	return points
}

func loadMatchingImages(api *serveAPI, mapGUID uuid.UUID, gpxData *gpx.GPX) []locatedImage {
	var document *document.Document

outer:
	for _, doc := range api.store.Documents {
		for _, m := range doc.Maps {
			if m.Resource.GUID == mapGUID {
				document = doc
				break outer
			}
		}
	}

	if document == nil {
		return nil
	}

	fmt.Println("holla!")

	absDuration := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	var locatedImages []locatedImage

	for _, gal := range document.Galleries {
		for _, img := range gal.Images {
			exifData, err := images.ReadEXIFFromFile(img.FilePath)
			if err != nil {
				log.Printf("could not load exif data: %s", err)
			}

			ts := exifData.Time
			var best gpx.GPXPoint

			for _, track := range gpxData.Tracks {
				for _, segment := range track.Segments {
					for _, point := range segment.Points {
						tsPoint := point.Timestamp.In(time.Local)
						tsBest := best.Timestamp.In(time.Local)
						diffCurr := absDuration(ts.Sub(tsBest))
						diffPoint := absDuration(ts.Sub(tsPoint))

						if diffPoint < diffCurr {
							best = point
						}
					}
				}
			}

			tsBest := best.Timestamp.In(time.Local)
			durBest := absDuration(ts.Sub(tsBest))

			if durBest < 120*time.Second {
				fmt.Printf("best diff: %s\n", durBest)
				locatedImages = append(
					locatedImages,
					locatedImage{
						URI: img.Resource.URI,
						LatLng: point{
							lat: best.Latitude,
							lon: best.Longitude,
						},
					},
				)
			}
		}
	}

	fmt.Printf("%#v\n", locatedImages)
	return locatedImages
}
