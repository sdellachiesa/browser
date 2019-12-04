// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

// Message represents the message received from and send to the client.
type Message struct {
	Fields   []string
	Stations []string
	Landuse  []string

	start time.Time
	end   time.Time
}

// filterQuery returns a 'SHOW TAG VALUES' query for filtering.
func (m *Message) filterQuery() ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		b := ql.ShowTagValues().From(m.Fields...).WithKeyIn("landuse", "snipeit_location_ref")

		var where ql.Querier
		if len(m.Stations) > 0 {
			where = ql.Eq(ql.Or(), "snipeit_location_ref", m.Stations...)
		}
		if len(m.Landuse) > 0 {
			where = ql.Eq(ql.Or(), "landuse", m.Landuse...)
		}

		b.Where(where, ql.And(), ql.TimeRange(time.Now().Add(-7*24*time.Hour), time.Now()))

		return b.Query()
	})
}

// seriesQuery returns one or multiple 'SELECT' queries for downloading
// time series data.
func (m *Message) seriesQuery() ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		var (
			buf  bytes.Buffer
			args []interface{}
		)
		for _, station := range m.Stations {
			columns := []string{"station", "landuse", "altitude", "latitude", "longitude"}
			columns = append(columns, m.Fields...)

			sb := ql.Select(columns...)
			sb.From(m.Fields...)
			sb.Where(
				ql.Eq(ql.And(), "snipeit_location_ref", station),
				ql.And(),
				ql.TimeRange(m.start, m.end),
			)
			sb.GroupBy("station,snipeit_location_ref")
			sb.OrderBy("time").ASC().TZ("Etc/GMT-1")

			q, arg := sb.Query()
			buf.WriteString(q)
			buf.WriteString(";")

			args = append(args, arg)
		}

		return buf.String(), args
	})
}
