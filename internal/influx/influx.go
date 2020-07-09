// Copyright 2020 Eurac Research. All rights reserved.

// Package influx provides the implementation of the browser.Database interface
// using InfluxDB as backend.
package influx

import (
	"context"
	"errors"
	"fmt"
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
