package geotrack

import (
	"bufio"
	"os"
	"time"

	"github.com/adrianmo/go-nmea"
)

func loadNMEATrack(trackFilePath string) (points []GPXPoint, err error) {
	f, err := os.Open(trackFilePath)
	if err != nil {
		return
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		if scanner.Err() != nil {
			return nil, scanner.Err()
		}

		sentence, err := nmea.Parse(scanner.Text())
		if err != nil {
			return nil, err
		}

		if sentence.DataType() == nmea.TypeRMC {
			rmc := sentence.(nmea.RMC)
			// We're only interested in "ACTIVE" status messages.
			if rmc.FFAMode != "A" {
				continue
			}

			// Adds 2000 to the date... I think this will be sufficient for life :)
			date := time.Date(
				2000+rmc.Date.YY, time.Month(rmc.Date.MM), rmc.Date.DD,
				rmc.Time.Hour, rmc.Time.Minute, rmc.Time.Second, 0, time.UTC,
			)

			points = append(points, GPXPoint{
				Lat:  rmc.Latitude,
				Lon:  rmc.Longitude,
				Time: date,
			})
		}
	}

	return
}
