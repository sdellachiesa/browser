// Copyright 2020 Eurac Research. All rights reserved.

package influx

import (
	"context"
	"testing"
	"time"

	"gitlab.inf.unibz.it/lter/browser"

	"github.com/google/go-cmp/cmp"
)

func TestQuery(t *testing.T) {
	var (
		dbName = "testdb"
		db     = &DB{Database: dbName}
		ctx    = context.Background()
	)

	location, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		in   *browser.Message
		want *browser.Stmt
	}{
		"empty": {
			&browser.Message{},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurement": {
			&browser.Message{Measurements: []string{"A"}},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, A FROM A WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurements": {
			&browser.Message{Measurements: []string{"A", "B"}},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, A, B FROM A, B WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"station": {
			&browser.Message{Stations: []string{"s1"}},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE snipeit_location_ref='s1' AND time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"stations": {
			&browser.Message{Stations: []string{"s1", "s2"}},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE snipeit_location_ref='s1' OR snipeit_location_ref='s2' AND time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"full": {
			&browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{"s1", "s2"},
				Start:        time.Date(2020, 1, 1, 0, 0, 0, 0, location),
				End:          time.Date(2020, 1, 1, 0, 0, 0, 0, location),
			},
			&browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, A, B, C FROM A, B, C WHERE snipeit_location_ref='s1' OR snipeit_location_ref='s2' AND time >= '2019-12-31T23:00:00Z' AND time <= '2020-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := db.Query(ctx, tc.in)

			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}

}
