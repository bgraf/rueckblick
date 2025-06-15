package geotrack

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/data/document"
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

func LoadTrack(trackFilePath string) (points []GPXPoint, err error) {
	ext := strings.ToLower(path.Ext(trackFilePath))
	if slices.Contains(config.GPXExtensions(), ext) {
		points, err = loadGPXTrack(trackFilePath)
	} else if slices.Contains(config.NMEAExtensions(), ext) {
		points, err = loadNMEATrack(trackFilePath)
	} else {
		return nil, fmt.Errorf("unknown track extension '%s'", ext)
	}

	if err != nil {
		return
	}

	return
}

// LoadTrackWithImages loads a track file from the given file path and correlates the documents images with
// the track's points.
func LoadTrackWithImages(doc *document.Document, trackFilePath string) (points []GPXPoint, images []GPXLocatedImage, err error) {
	points, err = LoadTrack(trackFilePath)
	if err != nil {
		return
	}

	images = findMatchingImages(doc, points)
	return
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
		}
	}

	return points[iBest], durBest
}
