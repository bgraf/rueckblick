package images

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"time"
)

var ErrNoExif = errors.New("no EXIF data")

type EXIFData struct {
	Time *time.Time
}

func ReadEXIFFromFile(path string) (*EXIFData, error) {
	command := exec.Command("exiftool", "-json", "-DateTimeOriginal", path)
	var buffer bytes.Buffer
	command.Stdout = &buffer
	if err := command.Run(); err != nil {
		return nil, err
	}

	var rawExifData []struct {
		DateTimeOriginal string `json:"DateTimeOriginal"`
	}

	if err := json.Unmarshal(buffer.Bytes(), &rawExifData); err != nil {
		return nil, err
	}

	if len(rawExifData[0].DateTimeOriginal) == 0 {
		return nil, ErrNoExif
	}

	dateTimeOriginal, err := time.Parse("2006:01:02 15:04:05", rawExifData[0].DateTimeOriginal)
	if err != nil {
		return nil, err
	}

	dateTimeOriginal = time.Date(
		dateTimeOriginal.Year(),
		dateTimeOriginal.Month(),
		dateTimeOriginal.Day(),
		dateTimeOriginal.Hour(),
		dateTimeOriginal.Minute(),
		dateTimeOriginal.Second(),
		dateTimeOriginal.Nanosecond(),
		time.Local,
	)

	return &EXIFData{Time: &dateTimeOriginal}, nil

}
