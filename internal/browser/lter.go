// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// Station defines metadata about a physical station.
type Station struct {
	ID        int64
	Name      string
	Landuse   string
	Altitude  int64
	Latitude  float64
	Longitude float64
}

func (s *Station) UnmarshalJSON(b []byte) error {
	var l snipeit.Location
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}

	s.ID = l.ID
	s.Name = l.Name
	s.Landuse = l.Currency
	s.Altitude, _ = strconv.ParseInt(l.Zip, 10, 64)
	s.Latitude, _ = strconv.ParseFloat(l.Address, 64)
	s.Longitude, _ = strconv.ParseFloat(l.Address2, 64)

	return nil
}

type Response struct {
	Stations []int64
	Fields   []string
	Landuse  []string
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type SeriesOptions struct {
	TimeRange
	FilterOptions
}

// NewSeriesOptionsFromForm parses the given request for form values and
// validates them.
func NewSeriesOptionsFromForm(r *http.Request) (*SeriesOptions, error) {
	start, err := time.Parse("2006-01-02", r.FormValue("startDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.Parse("2006-01-02", r.FormValue("endDate"))
	if err != nil {
		return nil, fmt.Errorf("error: could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	// Limit download of data to one year
	limit := time.Date(end.Year()-1, end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if start.Before(limit) {
		return nil, errors.New("error: time range is greater then a year")
	}

	if r.Form["fields"] == nil {
		return nil, errors.New("error: at least one field must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("error: at least one station must be given")
	}

	opts := &SeriesOptions{}
	opts.Fields = r.Form["fields"]
	opts.Stations = r.Form["stations"]
	opts.Landuse = r.Form["landuse"]

	// We need to shift the timerange of one hour since in influx we use UTC and in output we want
	// UTC+1.
	opts.Start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC).Add(-1 * time.Hour)
	opts.End = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.UTC).Add(-1 * time.Hour)

	return opts, nil
}

func (s *SeriesOptions) Query() (string, error) {
	qs := []string{}
	for _, f := range s.Stations {
		q := fmt.Sprintf("SELECT station,landuse,altitude,latitude,longitude,%s FROM %s WHERE %s AND time >= '%s' AND time <= '%s' GROUP BY station ORDER BY time ASC",
			strings.Join(s.Fields, ","),
			strings.Join(s.Fields, ","),
			fmt.Sprintf("snipeit_location_ref='%s'", f),
			s.TimeRange.Start.Format("2006-01-02T15:04:00Z"),
			s.TimeRange.End.Format("2006-01-02T15:04:00Z"),
		)
		log.Println(q)
		qs = append(qs, q)
	}

	return strings.Join(qs, ";"), nil
}
