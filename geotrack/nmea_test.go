package geotrack

import "testing"

func TestLoadNMEA(t *testing.T) {
	_, err := LoadNMEATrack("testdata/nmea.txt")
	if err != nil {
		t.Fail()
	}
}
