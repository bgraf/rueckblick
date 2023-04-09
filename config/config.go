package config

import (
	"log"

	"github.com/bgraf/rueckblick/data/geotrack"
	"github.com/spf13/viper"
)

var (
	KeyJournalDirectory = "journal.directory"
	KeyBuildDirectory   = "build.directory"
	KeyGeoHomeLat       = "geo.home.lat"
	KeyGeoHomeLon       = "geo.home.lon"
	KeyMapThreshold     = "geo.mapthreshold"
)

func HasJournalDirectory() bool {
	return viper.IsSet(KeyJournalDirectory)
}

func JournalDirectory() string {
	return viper.GetString(KeyJournalDirectory)
}

func HasBuildDirectory() bool {
	return viper.IsSet(KeyBuildDirectory)
}

func BuildDirectory() string {
	return viper.GetString(KeyBuildDirectory)
}

func DefaultPhotosDirectory() string {
	return "photos"
}

func DefaultGPXFile() string {
	return "track.gpx"
}

func DefaultPhotoWidth() int {
	return 2000
}

func DefaultPreviewWidth() int {
	return 600
}

func DefaultPreviewFilename() string {
	return "preview.jpg"
}

func DefaultPreviewJPEGQuality() int {
	return 95
}

func HomeCoords() geotrack.GPXPoint {
	if !viper.IsSet(KeyGeoHomeLat) || !viper.IsSet(KeyGeoHomeLon) {
		log.Fatalf("config: either %s or %s not set", KeyGeoHomeLat, KeyGeoHomeLon)
	}

	return geotrack.GPXPoint{
		Lat: viper.GetFloat64(KeyGeoHomeLat),
		Lon: viper.GetFloat64(KeyGeoHomeLon),
	}
}

func DefaultMapThreshold() float64 {
	return 50
}

func MapThreshold() float64 {
	if viper.IsSet(KeyMapThreshold) {
		return viper.GetFloat64(KeyMapThreshold)
	}

	return DefaultMapThreshold()
}
