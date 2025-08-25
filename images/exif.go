package images

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"time"

	"github.com/bgraf/rueckblick/geotrack"
	"github.com/bgraf/rueckblick/option"
)

var ErrNoExif = errors.New("no EXIF data")

type EXIFData struct {
	Time   option.Option[time.Time]
	LatLon option.Option[geotrack.GPXPoint]
}

func ReadEXIFFromFile(path string) (EXIFData, error) {
	command := exec.Command(
		"exiftool",
		"-json",
		"-n",
		"-DateTimeOriginal",
		"-GPSLatitude",
		"-GPSLatitudeRef",
		"-GPSLongitude",
		"-GPSLongitudeRef",
		path,
	)
	var buffer bytes.Buffer
	command.Stdout = &buffer
	if err := command.Run(); err != nil {
		return EXIFData{}, ErrNoExif
	}

	var rawExifData []struct {
		DateTimeOriginal string  `json:"DateTimeOriginal"`
		Lat              float64 `json:"GPSLatitude"`
		Lon              float64 `json:"GPSLongitude"`
	}

	if err := json.Unmarshal(buffer.Bytes(), &rawExifData); err != nil {
		return EXIFData{}, err
	}

	datetime := option.None[time.Time]()
	if len(rawExifData[0].DateTimeOriginal) > 0 {
		dateTimeOriginal, err := time.Parse("2006:01:02 15:04:05", rawExifData[0].DateTimeOriginal)
		if err == nil {
			datetime = option.Some(time.Date(
				dateTimeOriginal.Year(),
				dateTimeOriginal.Month(),
				dateTimeOriginal.Day(),
				dateTimeOriginal.Hour(),
				dateTimeOriginal.Minute(),
				dateTimeOriginal.Second(),
				dateTimeOriginal.Nanosecond(),
				time.Local,
			))
			//return nil, err
		}

	}

	latlon := option.None[geotrack.GPXPoint]()
	if rawExifData[0].Lat != 0 && rawExifData[0].Lon != 0 {
		t := time.Time{}
		if datetime.IsSome() {
			t = datetime.Get()
		}
		latlon = option.Some(geotrack.GPXPoint{
			Lat:  rawExifData[0].Lat,
			Lon:  rawExifData[0].Lon,
			Time: t,
		})
	}

	return EXIFData{
		Time:   datetime,
		LatLon: latlon,
	}, nil

}
