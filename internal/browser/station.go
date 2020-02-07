// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/json"
	"sort"
	"strconv"

	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// Station describes metadata about a physical LTER station.
// It is a custom type inorder to map custom SnipeIT location
// fields to their correct names.
type Station struct {
	ID           string
	Name         string
	Landuse      string
	Image        string
	Altitude     int64
	Latitude     float64
	Longitude    float64
	Measurements []string
}

// UnmarshalJSON is a custom JSON unmarshaler which maps SnipeIT
// location fields to the custom station field.
func (s *Station) UnmarshalJSON(b []byte) error {
	var l snipeit.Location
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}

	s.ID = strconv.FormatInt(l.ID, 10)
	s.Name = l.Name
	s.Landuse = l.Currency
	s.Image = l.Image
	s.Altitude, _ = strconv.ParseInt(l.Zip, 10, 64)
	s.Latitude, _ = strconv.ParseFloat(l.Address, 64)
	s.Longitude, _ = strconv.ParseFloat(l.Address2, 64)

	return nil
}

// Stations is a collection of LTER stations.
type Stations []*Station

// String converts stations to a JSON string.
func (s Stations) String() string {
	j, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}

	return string(j)
}

// Get returns the station by given id. If no station is found it
// will return nil and false for indicating that no station was
// found.
func (s Stations) Get(id string) (*Station, bool) {
	for _, station := range s {
		if id == station.ID {
			return station, true
		}
	}
	return nil, false
}

// WithMeasurements returns all  stations with at least one or
// more measurement.
func (s Stations) WithMeasurements() Stations {
	var stations Stations
	for _, station := range s {
		if len(station.Measurements) > 0 {
			stations = append(stations, station)
		}
	}
	return stations
}

// Landuse returns a sorted list of the landuse of all stations,
// removing duplicates.
func (s Stations) Landuse() []string {
	var l []string

	for _, station := range s {
		l = appendIfMissing(l, station.Landuse)
	}

	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	return l
}

// Measurements returns a sorted list of all measurements of all stations,
// removing duplicates.
func (s Stations) Measurements() []string {
	var v []string

	for _, station := range s {
		for _, f := range station.Measurements {
			v = appendIfMissing(v, f)
		}
	}

	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })

	return v
}

func appendIfMissing(slice []string, s string) []string {
	for _, el := range slice {
		if el == s {
			return slice
		}
	}
	return append(slice, s)
}
