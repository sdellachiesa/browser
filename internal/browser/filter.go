// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

type QueryType int

const (
	UpdateQuery QueryType = iota
	SeriesQuery
)

// Filter holds lists of specific properties like Fields, Stations
// or Landuse for filtering data on the backend side.
type Filter struct {
	Fields   []string
	Stations []string
	Landuse  []string

	start time.Time
	end   time.Time

	qType QueryType
}

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

func (f *Filter) seriesQuery() (string, []interface{}) {
	var (
		buf  bytes.Buffer
		args []interface{}
	)
	for _, station := range f.Stations {
		columns := []string{"station", "landuse", "altitude", "latitude"}
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
