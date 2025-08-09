package geotrack

import (
	"encoding/json"
	"time"
)

type GPXPoint struct {
	Lat, Lon float64
	Time     time.Time
}

func (p GPXPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{p.Lat, p.Lon})
}
