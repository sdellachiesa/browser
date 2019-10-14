// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

// QueryType denotes specific type of query.
type QueryType int

const (
	// UpdateQuery denotes a 'SHOW TAG QUERY' for filtering.
	UpdateQuery QueryType = iota
	// SeriesQuery denotes a 'SELECT' query.
	SeriesQuery
)

// Filter holds a list of specific properties for filtering and
// downloading data from InfluxDB.
type Filter struct {
	Fields   []string
	Stations []string
	Landuse  []string

	start time.Time
	end   time.Time

	qType QueryType
}

// Query implements the ql.Querier interface.
func (f *Filter) Query() (string, []interface{}) {
	switch f.qType {
	case UpdateQuery:
		return f.updateQuery()
	case SeriesQuery:
		return f.seriesQuery()
	default:
		return "", nil
	}
}

// updateQuery returns a 'SHOW TAG VALUES' query for filtering.
func (f *Filter) updateQuery() (string, []interface{}) {
	b := ql.ShowTagValues().From(f.Fields...).WithKeyIn("landuse", "snipeit_location_ref")
	if len(f.Stations) > 0 {
		b.Where(ql.Eq(ql.Or(), "snipeit_location_ref", f.Stations...))
	}
	if len(f.Landuse) > 0 {
		b.Where(ql.Eq(ql.Or(), "landuse", f.Landuse...))
	}
	return b.Query()
}

// seriesQuery returns one or multiple 'SELECT' queries for downloading
// time series data.
func (f *Filter) seriesQuery() (string, []interface{}) {
	var (
		buf  bytes.Buffer
		args []interface{}
	)
	for _, station := range f.Stations {
		columns := []string{"station", "landuse", "altitude", "latitude", "longitude"}
		columns = append(columns, f.Fields...)

		sb := ql.Select(columns...)
		sb.From(f.Fields...)
		sb.Where(
			ql.Eq(ql.And(), "snipeit_location_ref", station),
			ql.And(),
			ql.TimeRange(f.start, f.end),
		)
		sb.GroupBy("station")
		sb.OrderBy("time").ASC()

		q, arg := sb.Query()
		buf.WriteString(q)
		buf.WriteString(";")

		args = append(args, arg)
	}

	return buf.String(), args
}
