// Copyright 2020 Eurac Research. All rights reserved.

// Package snipeit provides
package snipeit

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"

	"github.com/euracresearch/go-snipeit"
	client "github.com/influxdata/influxdb1-client/v2"
)

// Ensure SnipeITService implements browser.Metadata.
var _ browser.Metadata = &SnipeITService{}

// SnipeITService represents a service for retriving metadata stored in SnipeIT.
// Unfortunately not all metadata is currently present in SnipeIT, like for
// example all  measurements of a station. In order to get such metadata we have
// to additionally query InfluxDB.
type SnipeITService struct {
	client *snipeit.Client

	db       client.Client
	database string
}

// NewSnipeITService returns a new instance of SnipeITService.
func NewSnipeITService(baseurl, token string, db client.Client, database string) (*SnipeITService, error) {
	c, err := snipeit.NewClient(baseurl, token)
	if err != nil {
		return nil, err
	}

	return &SnipeITService{
		client:   c,
		db:       db,
		database: database,
	}, nil
}

// Stations implements browser.Metadata.
func (s *SnipeITService) Stations(ctx context.Context, m *browser.Message) (browser.Stations, error) {
	var where ql.Querier
	if len(m.Stations) > 0 {
		where = ql.Eq(ql.Or(), "snipeit_location_ref", m.Stations...)
	}
	if len(m.Landuse) > 0 {
		where = ql.Eq(ql.Or(), "landuse", m.Landuse...)
	}

	q, _ := ql.ShowTagValues().From(m.Measurements...).WithKeyIn("snipeit_location_ref").Where(where).Query()
	resp, err := s.db.Query(client.NewQuery(q, s.database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	measurements := make(map[string][]string)
	for _, result := range resp.Results {
		for _, s := range result.Series {
			for _, v := range s.Values {
				id := v[1].(string)
				measurements[id] = append(measurements[id], s.Name)
			}
		}
	}

	stations, err := s.stations(m, measurements)
	if err != nil {
		return nil, err
	}

	return stations, nil
}

func (s *SnipeITService) stations(m *browser.Message, measurements map[string][]string) (browser.Stations, error) {
	opts := &snipeit.LocationOptions{
		Search: "LTER",
		Limit:  100,
	}

	locations, _, err := s.client.Locations(opts)
	if err != nil {
		return nil, err
	}

	var stations browser.Stations
	for _, l := range locations {
		if l.Name == "LTER" {
			continue
		}

		id := strconv.FormatInt(l.ID, 10)
		elevation, _ := strconv.ParseInt(l.Zip, 10, 64)
		latitude, _ := strconv.ParseFloat(l.Address, 64)
		longitude, _ := strconv.ParseFloat(l.Address2, 64)

		if !inArray(id, m.Stations) {
			continue
		}

		// Only consider stations with measurements.
		ms, ok := measurements[id]
		if !ok {
			continue
		}

		stations = append(stations, &browser.Station{
			Name:         l.Name,
			ID:           id,
			Landuse:      l.Currency,
			Image:        l.Image,
			Elevation:    elevation,
			Latitude:     latitude,
			Longitude:    longitude,
			Measurements: ms,
		})
	}

	// Sort stations by name.
	sort.Slice(stations, func(i, j int) bool {
		return stations[i].Name < stations[j].Name
	})

	return stations, nil
}

// inArray checks if the given s is in the given slice.
// If the given slice is empty true will be returned.
func inArray(s string, a []string) bool {
	if len(a) == 0 {
		return true
	}

	for _, v := range a {
		if v == s {
			return true
		}
	}

	return false
}
