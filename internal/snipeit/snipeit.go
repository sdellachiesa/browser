// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package snipeit provides a service for retriving information stored in SnipeIT.
package snipeit

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/go-snipeit"
)

// Ensure StationService implements browser.StationService.
var _ browser.StationService = &StationService{}

// StationService represents a service for retriving information stored in
// SnipeIT.
type StationService struct {
	client *snipeit.Client
}

// NewStationService returns a new instance of SnipeITService.
func NewStationService(baseurl, token string) (*StationService, error) {
	c, err := snipeit.NewClient(baseurl, token)
	if err != nil {
		return nil, err
	}

	return &StationService{
		client: c,
	}, nil
}

// Station implements browser.StationService.
func (s *StationService) Station(ctx context.Context, id int64) (*browser.Station, error) {
	location, resp, err := s.client.Location(id)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SnipeIT API returned an error: %s", resp.Status)
	}

	station, err := parseStation(location)
	if err != nil {
		return nil, err
	}

	return station, nil
}

// parseStation parses a browser.Station from a snipeit.Location.
func parseStation(l *snipeit.Location) (*browser.Station, error) {
	//	id := strconv.FormatInt(l.ID, 10)
	elevation, err := strconv.ParseInt(l.Zip, 10, 64)
	if err != nil {
		return nil, err
	}
	latitude, err := strconv.ParseFloat(l.Address, 64)
	if err != nil {
		return nil, err
	}
	longitude, err := strconv.ParseFloat(l.Address2, 64)
	if err != nil {
		return nil, err
	}

	return &browser.Station{
		Name:      l.Name,
		ID:        l.ID,
		Landuse:   l.Currency,
		Image:     l.Image,
		Dashboard: l.City,
		Elevation: elevation,
		Latitude:  latitude,
		Longitude: longitude,
	}, nil
}

// Stations implements browser.StationService.
func (s *StationService) Stations(ctx context.Context) (browser.Stations, error) {
	opts := &snipeit.LocationOptions{
		Search: "LTER",
		Limit:  100,
	}

	locations, resp, err := s.client.Locations(opts)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SnipeIT API returned an error: %s", resp.Status)
	}

	var stations browser.Stations
	for _, l := range locations {
		if strings.EqualFold(l.Name, "LTER") {
			continue
		}

		station, err := parseStation(l)
		if err != nil {
			continue
		}

		stations = append(stations, station)
	}

	// Sort stations by name.
	sort.Slice(stations, func(i, j int) bool {
		return stations[i].Name < stations[j].Name
	})

	return stations, nil
}
