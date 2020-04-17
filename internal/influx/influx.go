// Copyright 2020 Eurac Research. All rights reserved.

// Package influx provides the implementation of the browser.Database interface
// using InfluxDB as backend.
package influx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"

	client "github.com/influxdata/influxdb1-client/v2"
)

// Guarantee we implement browser.Series.
var _ browser.Database = &DB{}

// DB holds information for communicating with InfluxDB.
type DB struct {
	Client   client.Client
	Database string
}

// NewDB returns a new instance of DB.
func NewDB(client client.Client, database string) *DB {
	return &DB{
		Client:   client,
		Database: database,
	}
}

// TODO: new API.
func (db *DB) Series(ctx context.Context, m *browser.Message) (browser.TimeSeries, error) {
	return nil, errors.New("not yet implemented")
}

func (db *DB) Query(ctx context.Context, m *browser.Message) *browser.Stmt {
	c := []string{"station", "landuse", "altitude as elevation", "latitude", "longitude"}
	c = append(c, m.Measurements...)

	// Data in influx is UTC but LTER data is UTC+1 therefor
	// we need to adapt start and end times. It will shift the start
	// time to -1 hour and will set the end time to 22:59:59 in order to
	// capture a full day.
	start := m.Start.Add(-1 * time.Hour)
	end := time.Date(m.End.Year(), m.End.Month(), m.End.Day(), 22, 59, 59, 59, time.UTC)

	q, _ := ql.Select(c...).From(m.Measurements...).Where(
		ql.Eq(ql.Or(), "snipeit_location_ref", m.Stations...),
		ql.And(),
		ql.TimeRange(start, end),
	).OrderBy("time").ASC().TZ("Etc/GMT-1").Query()

	return &browser.Stmt{
		Query:    q,
		Database: db.Database,
	}
}

// exec executes the given ql query and returns a response.
func (db *DB) exec(q ql.Querier) (*client.Response, error) {
	query, _ := q.Query()

	resp, err := db.Client.Query(client.NewQuery(query, db.Database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	return resp, nil
}

func seriesQuery(m *browser.Message) ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		var (
			buf  bytes.Buffer
			args []interface{}
		)

		// Data in influx is UTC but LTER data is UTC+1 therefor
		// we need to adapt start and end times. It will shift the start
		// time to -1 hour and will set the end time to 22:59:59 in order to
		// capture a full day.
		start := m.Start.Add(-1 * time.Hour)
		end := time.Date(m.End.Year(), m.End.Month(), m.End.Day(), 22, 59, 59, 59, time.UTC)

		for _, station := range m.Stations {
			columns := []string{"station", "landuse", "altitude as elevation", "latitude", "longitude"}
			columns = append(columns, m.Measurements...)

			sb := ql.Select(columns...)
			sb.From(m.Measurements...)
			sb.Where(
				ql.Eq(ql.And(), "snipeit_location_ref", station),
				ql.And(),
				ql.TimeRange(start, end),
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

func (db *DB) units(msg *browser.Message, h []string) ([]string, error) {
	units := make([]string, len(h))

	q := ql.ShowTagValues().From(msg.Measurements...).WithKeyIn("unit").Where(ql.TimeRange(msg.Start, msg.End))
	resp, err := db.exec(q)
	if err != nil {
		return units, err
	}

	m := make(map[string]string)
	for _, result := range resp.Results {
		for _, serie := range result.Series {
			for _, value := range serie.Values {
				m[serie.Name] = value[1].(string)
			}
		}
	}

	for i, v := range h {
		u, ok := m[v]
		if !ok {
			continue
		}

		units[i] = u
	}

	return units, nil
}

// key is used as map key for sorting and grouping the map entries.
type key struct {
	station string

	// timestamp is a UNIX epoch and not of a time.Time since
	// the later should be avoid for map keys.
	// Read https://golang.org/src/time/time.go?#L101
	timestamp int64
}

// Time returns the key's unix timestamp as time.Time.
func (k key) Time() time.Time {
	timeLoc := time.FixedZone("UTC+1", 60*60)
	return time.Unix(k.timestamp, 0).In(timeLoc)
}

// Next returns a new key with it's timestamp modified by the given duration.
func (k key) Next(d time.Duration) key {
	return key{k.station, k.Time().Add(d).Unix()}
}

// TODO(m): This is the current version for getting a timeseries as CSV. It is
// slow on big data queries and not flexible enough to support multiple types of
// CSV formats.
func (db *DB) SeriesV1(ctx context.Context, m *browser.Message) ([][]string, error) {
	resp, err := db.exec(seriesQuery(m))
	if err != nil {
		return nil, err
	}

	var (
		table  = make(map[key][]string)
		keys   = []key{}
		header = []string{}
	)
	for _, result := range resp.Results {
		for i, serie := range result.Series {
			if i == 0 {
				header = serie.Columns
			}

			for _, value := range serie.Values {
				ts, err := time.ParseInLocation(time.RFC3339, value[0].(string), time.UTC)
				if err != nil {
					log.Printf("cannot convert timestamp: %v. skipping.", err)
					continue
				}

				k := key{serie.Tags["snipeit_location_ref"], ts.Unix()}

				column, ok := table[k]
				if !ok {
					keys = append(keys, k)

					// Initialize column and fill it with NaN's.
					column = make([]string, len(value))
					for i := range column {
						column[i] = "NaN"
					}
				}

				for i := range value {
					v := value[i]
					if v == nil {
						continue
					}

					// The value at index 0 corresponds to the timestamp.
					if i == 0 {
						v = ts.Format("2006-01-02 15:04:05")
					}

					column[i] = fmt.Sprint(v)

				}

				table[k] = column
			}
		}
	}

	if len(keys) == 0 {
		return nil, browser.ErrDataNotFound
	}

	// Sort by timesstamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].timestamp < keys[j].timestamp
	})

	// Sort by station,timestamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].station < keys[j].station
	})

	units, _ := db.units(m, header)
	rows := [][]string{header, units}
	last := keys[len(keys)-1]

	for _, k := range keys {
		c, ok := table[k]
		if !ok {
			continue
		}
		rows = append(rows, c)

		// Fill up missing timestamps in order to have a continuous timerange
		// otherwise Researchers cannot work with the data and will complain.
		// What a bummer. The interval of 'RAW' data in LTER is always 15 Minutes.
		// See: https://gitlab.inf.unibz.it/lter/browser/issues/10
		next := k.Next(15 * time.Minute)
		for {
			// Check if there is a record for the next timestamp if yes
			// break out of the for loop. Otherwise stay in the loop and
			// fill up missing rows until the next real entry.
			_, ok := table[next]
			if ok || next.Time().After(last.Time()) {
				break
			}

			// Initialize column and fill it with NaN's.
			column := make([]string, len(c))
			for i := range column {
				column[i] = "NaN"
			}
			column[0] = next.Time().Format("2006-01-02 15:04:05")
			column[1] = c[1]

			rows = append(rows, column)
			next = next.Next(15 * time.Minute)
		}
	}

	return rows, nil
}
