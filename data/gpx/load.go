package gpx

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bgraf/rueckblick/data/document"
	"github.com/tkrajina/gpxgo/gpx"
)

type GPXLocatedImage struct {
	URI    string
	LatLng GPXPoint
}

type GPXPoint struct {
	Lat, Lon float64
	Time     time.Time
}

func (p GPXPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.Lat, p.Lon})
}

func LoadGPX(doc *document.Document, gpxFilePath string) ([]GPXPoint, []GPXLocatedImage, error) {
	gpxData, err := gpx.ParseFile(gpxFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read GPX file: %w", err)
	}

	track := readGPXTrack(gpxData)
	locatedImages := findMatchingImages(doc, track)

	return track, locatedImages, nil
}

func readGPXTrack(gpxFile *gpx.GPX) []GPXPoint {
	var points []GPXPoint

	for _, track := range gpxFile.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				points = append(points, GPXPoint{Lat: p.Latitude, Lon: p.Longitude, Time: p.Timestamp})
			}
		}
	}

	return points
}

func findMatchingImages(doc *document.Document, points []GPXPoint) []GPXLocatedImage {
	var locatedImages []GPXLocatedImage

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
					GPXLocatedImage{
						URI:    img.Resource.URI,
						LatLng: GPXPoint{Lat: nearest.Lat, Lon: nearest.Lon},
					},
				)
			}
		}
	}

	return locatedImages
}

func findClosestPointInTime(points []GPXPoint, targetTime time.Time) (GPXPoint, time.Duration) {
	absDuration := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	durBest := time.Duration(1 << 62)
	iBest := 0
	for i := 0; i < len(points); i++ {
		durI := absDuration(targetTime.Sub(points[i].Time))
		if durI <= durBest {
			durBest = durI
			iBest = i
		} else {
			return points[i-1], durBest
		}
	}

	return points[iBest], durBest
}
