// Copyright 2020 Eurac Research. All rights reserved.

package influx

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
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
				Start:        time.Date(2020, 1, 1, 0, 0, 0, 0, browser.Location),
				End:          time.Date(2020, 1, 1, 0, 0, 0, 0, browser.Location),
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
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSeries(t *testing.T) {
	c := &mock.InfluxClient{}
	db := &DB{Client: c, Database: "testdb"}
	ctx := context.Background()

	// In test we use always the same message since we use a mock implementation
	// of the influx client interface which simple returns a client.Reponse from
	// a give JSON file.
	testMessage := &browser.Message{
		Measurements: []string{"air_rh_avg", "air_t_avg", "snow_height"},
		Stations:     []string{"39", "4"},
		Start:        time.Date(2020, 5, 4, 0, 0, 0, 0, browser.Location),
		End:          time.Date(2020, 5, 4, 0, 0, 0, 0, browser.Location),
	}

	testCases := map[string]struct {
		in      *browser.Message
		queryFn func(q client.Query) (*client.Response, error)
		want    browser.TimeSeries
	}{
		"nil": {
			nil,
			nil,
			nil,
		},
		"missing points": {
			in:      testMessage,
			queryFn: queryTestHelper(t, "missing.json"),
			want: browser.TimeSeries{
				&browser.Measurement{
					Label:       "air_rh_avg",
					Station:     "b1",
					Aggregation: "avg",
					Landuse:     "me",
					Unit:        "%",
					Elevation:   990,
					Latitude:    46.6612188656,
					Longitude:   10.5902491243,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T00:15:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T00:30:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T00:45:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T01:00:00+01:00", 48.98),
						testPoint(t, "2020-05-04T01:15:00+01:00", 52.53),
						testPoint(t, "2020-05-04T01:30:00+01:00", 53.07),
						testPoint(t, "2020-05-04T01:45:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T02:00:00+01:00", 54.25),
						testPoint(t, "2020-05-04T02:15:00+01:00", 57.86),
						testPoint(t, "2020-05-04T02:30:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T02:45:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T03:00:00+01:00", 59.52),
						testPoint(t, "2020-05-04T03:15:00+01:00", 59.41),
					},
				},
			},
		},
		"multiple measurements": {
			in:      testMessage,
			queryFn: queryTestHelper(t, "multiple.json"),
			want: browser.TimeSeries{
				&browser.Measurement{
					Label:       "air_rh_avg",
					Station:     "b1",
					Aggregation: "avg",
					Landuse:     "me",
					Unit:        "%",
					Elevation:   990,
					Latitude:    46.6612188656,
					Longitude:   10.5902491243,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T00:15:00+01:00", 48.1),
						testPoint(t, "2020-05-04T00:30:00+01:00", 45.6),
						testPoint(t, "2020-05-04T00:45:00+01:00", 46.93),
						testPoint(t, "2020-05-04T01:00:00+01:00", 48.98),
					},
				},
				&browser.Measurement{
					Label:       "air_rh_avg",
					Station:     "b2",
					Aggregation: "avg",
					Landuse:     "me",
					Unit:        "%",
					Elevation:   1490,
					Latitude:    46.6862577024,
					Longitude:   10.5798451965,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 44.91),
						testPoint(t, "2020-05-04T00:15:00+01:00", 44.54),
						testPoint(t, "2020-05-04T00:30:00+01:00", 45.43),
						testPoint(t, "2020-05-04T00:45:00+01:00", 47.45),
						testPoint(t, "2020-05-04T01:00:00+01:00", 49.49),
					},
				},
				&browser.Measurement{
					Label:       "air_t_avg",
					Station:     "b1",
					Aggregation: "avg",
					Landuse:     "me",
					Unit:        "deg c",
					Elevation:   990,
					Latitude:    46.6612188656,
					Longitude:   10.5902491243,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 10.05),
						testPoint(t, "2020-05-04T00:15:00+01:00", 9.46),
						testPoint(t, "2020-05-04T00:30:00+01:00", 9.61),
						testPoint(t, "2020-05-04T00:45:00+01:00", 9.72),
						testPoint(t, "2020-05-04T01:00:00+01:00", 9.02),
					},
				},
				&browser.Measurement{
					Label:       "air_t_avg",
					Station:     "b2",
					Aggregation: "avg",
					Landuse:     "me",
					Unit:        "deg c",
					Elevation:   1490,
					Latitude:    46.6862577024,
					Longitude:   10.5798451965,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 7.379),
						testPoint(t, "2020-05-04T00:15:00+01:00", 6.933),
						testPoint(t, "2020-05-04T00:30:00+01:00", 6.783),
						testPoint(t, "2020-05-04T00:45:00+01:00", 6.53),
					},
				},
				&browser.Measurement{
					Label:       "snow_height",
					Station:     "b1",
					Aggregation: "smp",
					Landuse:     "me",
					Unit:        "",
					Elevation:   990,
					Latitude:    46.6612188656,
					Longitude:   10.5902491243,
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 0.723),
						testPoint(t, "2020-05-04T00:15:00+01:00", 0.716),
						testPoint(t, "2020-05-04T00:30:00+01:00", 0.717),
						testPoint(t, "2020-05-04T00:45:00+01:00", 0.72),
						testPoint(t, "2020-05-04T01:00:00+01:00", 0.724),
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			c.QueryFn = tc.queryFn
			got, _ := db.Series(ctx, tc.in)

			diff := cmp.Diff(tc.want, got, cmp.Comparer(func(x, y float64) bool {
				return (math.IsNaN(x) && math.IsNaN(y)) || x == y
			}))
			if diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func testPoint(t *testing.T, s string, value float64) *browser.Point {
	t.Helper()

	ts, err := time.ParseInLocation(time.RFC3339, s, browser.Location)
	if err != nil {
		t.Fatal(err)
	}
	return &browser.Point{
		Timestamp: ts,
		Value:     value,
	}
}

func queryTestHelper(t *testing.T, filename string) func(q client.Query) (*client.Response, error) {
	t.Helper()

	return func(q client.Query) (*client.Response, error) {
		f, err := os.Open(filepath.Join("testdata", filename))
		if err != nil {
			return nil, err
		}
		defer f.Close()

		dec := json.NewDecoder(f)
		dec.UseNumber()

		var resp *client.Response
		if err := dec.Decode(&resp); err != nil {
			return nil, err
		}

		return resp, nil
	}
}
