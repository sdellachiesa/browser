// Copyright 2021 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package browser

import (
	"context"
	"encoding/json"
	"sort"
)

// Station represents a meteorological station of the LTER project.
type Station struct {
	ID        int64
	Name      string
	Landuse   string
	Elevation int64
	Latitude  float64
	Longitude float64
	Image     string
	Dashboard string
}

// StationService represents a service for retriving stations.
type StationService interface {
	// Station returns the station by the given id or an error.
	Station(ctx context.Context, id int64) (*Station, error)

	// Stations retrieves metadata about all stations.
	Stations(ctx context.Context) (Stations, error)
}

// Stations represents a group of meteorological stations.
type Stations []*Station

// String converts stations to a JSON string.
func (s Stations) String() string {
	j, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}

	return string(j)
}

// Landuse returns a sorted list of the landuse for all stations, removing
// duplicates.
func (s Stations) Landuse() []string {
	var l []string

	for _, station := range s {
		l = AppendStringIfMissing(l, station.Landuse)
	}

	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	return l
}

// AppendStringIfMissing will append the given string to the given slice if it
// is missing.
func AppendStringIfMissing(slice []string, s string) []string {
	for _, el := range slice {
		if el == s {
			return slice
		}
	}
	return append(slice, s)
}
