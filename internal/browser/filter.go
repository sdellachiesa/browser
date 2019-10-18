// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

// Filter holds a list of specific properties for filtering and
// downloading data from InfluxDB.
type Filter struct {
	Fields   []string
	Stations []string
	Landuse  []string

	start time.Time
	end   time.Time
}

// filterQuery returns a 'SHOW TAG VALUES' query for filtering.
func (f *Filter) filterQuery() ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		b := ql.ShowTagValues().From(f.Fields...).WithKeyIn("landuse", "snipeit_location_ref")

		var where ql.Querier
		if len(f.Stations) > 0 {
			where = ql.Eq(ql.Or(), "snipeit_location_ref", f.Stations...)
		}
		if len(f.Landuse) > 0 {
			where = ql.Eq(ql.Or(), "landuse", f.Landuse...)
		}

		b.Where(where, ql.And(), ql.TimeRange(time.Now().Add(-7*24*time.Hour), time.Now()))

		return b.Query()
	})
}

// seriesQuery returns one or multiple 'SELECT' queries for downloading
// time series data.
func (f *Filter) seriesQuery() ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
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
			sb.OrderBy("time").ASC().TZ("Etc/GMT-1")

			q, arg := sb.Query()
			buf.WriteString(q)
			buf.WriteString(";")

			args = append(args, arg)
		}

		return buf.String(), args
	})
}
