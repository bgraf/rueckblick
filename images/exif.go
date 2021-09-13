package images

import (
	"fmt"
	"os"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

type EXIFData struct {
	Time *time.Time
}

func ReadEXIFFromFile(path string) (*EXIFData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open failed: %w", err)
	}

	x, err := exif.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("exif.Decode failed: %w", err)
	}

	exif := &EXIFData{}

	if tm, err := x.DateTime(); err == nil {
		exif.Time = &tm
	}

	return exif, nil
}
