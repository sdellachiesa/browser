// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/json"
	"strconv"

	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// Station describes metadata about a physical LTER station.
// It is a custom type inorder to map custom SnipeIT location
// fields to their correct names.
type Station struct {
	ID        string
	Name      string
	Landuse   string
	Image     string
	Altitude  int64
	Latitude  float64
	Longitude float64
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
