package geotrack

import (
	"fmt"

	"github.com/tkrajina/gpxgo/gpx"
)

func loadGPXTrack(trackFilePath string) (points []GPXPoint, err error) {
	gpxData, err := gpx.ParseFile(trackFilePath)
	if err != nil {
		return nil, fmt.Errorf("read GPX file: %w", err)
	}

	for _, track := range gpxData.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, GPXPoint{Lat: p.Latitude, Lon: p.Longitude, Time: p.Timestamp})
			}
		}
	}

	return
}
