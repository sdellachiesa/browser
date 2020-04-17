// Copyright 2020 Eurac Research. All rights reserved.

package influx

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/mock"

	"github.com/google/go-cmp/cmp"
	client "github.com/influxdata/influxdb1-client/v2"
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

func TestSeriesV1(t *testing.T) {
	ctx := context.Background()

	location, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	c := &mock.InfluxClient{}
	db := &DB{
		Client:   c,
		Database: "testdb",
	}

	testCases := map[string]struct {
		in          *browser.Message
		queryFn     func(q client.Query) (*client.Response, error)
		wantCSVFile string
	}{
		"empty": {
			&browser.Message{},
			queryTestHelper(""),
			"",
		},
		"day": {
			&browser.Message{
				Measurements: []string{"air_rh_avg", "air_t_avg", "snow_height"},
				Stations:     []string{"39", "4"},
				Start:        time.Date(2020, 5, 4, 0, 0, 0, 0, location),
				End:          time.Date(2020, 5, 4, 0, 0, 0, 0, location),
			},
			queryTestHelper("day.json"),
			"day.csv",
		},
		"days": {
			&browser.Message{
				Measurements: []string{"air_rh_avg", "air_t_avg", "snow_height"},
				Stations:     []string{"39", "4"},
				Start:        time.Date(2020, 5, 2, 0, 0, 0, 0, location),
				End:          time.Date(2020, 5, 4, 0, 0, 0, 0, location),
			},
			queryTestHelper("days.json"),
			"days.csv",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			c.QueryFn = tc.queryFn

			got, _ := db.SeriesV1(ctx, tc.in)

			f, _ := os.Open(filepath.Join("testdata", tc.wantCSVFile))
			r := csv.NewReader(f)
			want, _ := r.ReadAll()

			diff := cmp.Diff(want, got)
			if diff != "" {
				t.Fatalf("SeriesV!1() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func queryTestHelper(filename string) func(q client.Query) (*client.Response, error) {
	return func(q client.Query) (*client.Response, error) {
		if strings.HasPrefix(q.Command, "SHOW TAG") {
			filename = "units.json"
		}

		b, err := ioutil.ReadFile(filepath.Join("testdata", filename))
		if err != nil {
			return nil, err
		}

		var resp *client.Response
		if err := json.Unmarshal(b, &resp); err != nil {
			return nil, err
		}

		return resp, nil
	}
}
