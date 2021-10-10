package config

import "github.com/spf13/viper"

var (
	KeyJournalDirectory = "journal.directory"
)

func HasJournalDirectory() bool {
	return viper.IsSet(KeyJournalDirectory)
}

func JournalDirectory() string {
	return viper.GetString(KeyJournalDirectory)
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
