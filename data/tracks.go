package data

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/bgraf/rueckblick/config"
	"github.com/bgraf/rueckblick/geotrack"
	"github.com/jftuga/geodist"
)

type GPXLocatedImage struct {
	URI      string
	ThumbURI string
	LatLng   geotrack.GPXPoint
}

func LoadTrack(trackFilePath string) (points []geotrack.GPXPoint, err error) {
	ext := strings.ToLower(path.Ext(trackFilePath))
	if slices.Contains(config.GPXExtensions(), ext) {
		points, err = geotrack.LoadGPXTrack(trackFilePath)
	} else if slices.Contains(config.NMEAExtensions(), ext) {
		points, err = geotrack.LoadNMEATrack(trackFilePath)
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
func LoadTrackWithImages(doc *Document, trackFilePath string) (points []geotrack.GPXPoint, images []GPXLocatedImage, err error) {
	points, err = LoadTrack(trackFilePath)
	if err != nil {
		return
	}

	images = findMatchingImages(doc, points)
	return
}

func findMatchingImages(doc *Document, points []geotrack.GPXPoint) []GPXLocatedImage {
	var locatedImages []GPXLocatedImage

	for _, gal := range doc.Galleries {
		for _, img := range gal.Images {
			if img.Timestamp.IsNone() {
				continue
			}
			targetTime := img.Timestamp.Get()
			nearest, duration := findClosestPointInTime(points, targetTime)

			if duration < 120*time.Second {
				if img.LatLon.IsSome() {
					fmt.Printf("Found image with own coords: %#v %#v\n", img.LatLon.Get(), nearest)

					_, dkm := geodist.HaversineDistance(
						geodist.Coord{
							Lat: img.LatLon.Get().Lat,
							Lon: img.LatLon.Get().Lon,
						},
						geodist.Coord{
							Lat: nearest.Lat,
							Lon: nearest.Lon,
						},
					)
					fmt.Printf("distance: %f\n", dkm)

				}
				locatedImages = append(
					locatedImages,
					GPXLocatedImage{
						URI:      img.Resource.URI,
						ThumbURI: img.ThumbResource.URI,
						LatLng:   geotrack.GPXPoint{Lat: nearest.Lat, Lon: nearest.Lon},
					},
				)
			}
		}
	}

	return locatedImages
}

func findClosestPointInTime(points []geotrack.GPXPoint, targetTime time.Time) (geotrack.GPXPoint, time.Duration) {
	absDuration := func(d time.Duration) time.Duration {
		if d < 0 {
			return -d
		}
		return d
	}

	durBest := time.Duration(1 << 62)
	iBest := 0
	for i := range points {
		durI := absDuration(targetTime.Sub(points[i].Time))
		if durI <= durBest {
			durBest = durI
			iBest = i
		}
	}

	return points[iBest], durBest
}
