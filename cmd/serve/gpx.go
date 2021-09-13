package serve

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tkrajina/gpxgo/gpx"
)

func (api *serveAPI) ServeGPX(c *gin.Context) {
	key := c.Param("GUID")
	gpxFilePath, ok := api.rewriter.PathFromID(key)
	if !ok {
		c.String(http.StatusNotFound, "not found")
		return
	}

	track, err := readGPXTrack(gpxFilePath)
	if err != nil {
		_ = c.Error(err)
		c.String(http.StatusInternalServerError, "error during GPX reading")
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"track": track,
		},
	)
}

type point struct {
	lat, lon float64
}

func (p point) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.lat, p.lon})
}

func readGPXTrack(path string) ([]point, error) {
	gpxFile, err := gpx.ParseFile(path)
	if err != nil {
		return nil, err
	}

	points := []point{}

	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, point{p.Latitude, p.Longitude})
			}
		}
	}

	return points, nil
}
