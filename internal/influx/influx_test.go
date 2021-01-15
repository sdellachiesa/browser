// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package influx

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/mock"

	"github.com/google/go-cmp/cmp"
	client "github.com/influxdata/influxdb1-client/v2"
)

func TestQuery(t *testing.T) {
	dbName := "testdb"

	testCases := map[string]struct {
		in   *browser.SeriesFilter
		ctx  context.Context
		want *browser.Stmt
	}{
		"empty": {
			in:  &browser.SeriesFilter{},
			ctx: context.Background(),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurement": {
			in:  &browser.SeriesFilter{Groups: []browser.Group{browser.WindSpeed}},
			ctx: context.Background(),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, wind_speed_avg, wind_speed_max FROM wind_speed_avg, wind_speed_max WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"subgroup": {
			in:  &browser.SeriesFilter{Groups: []browser.Group{browser.WindSpeedAvg}},
			ctx: context.Background(),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, wind_speed_avg FROM wind_speed_avg WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurements_public_false": {
			in:  &browser.SeriesFilter{Groups: []browser.Group{browser.AirTemperature, browser.SoilTemperature}},
			ctx: createContext(t, browser.Public, false),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, air_t_avg FROM air_t_avg WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurements_public_true": {
			in:  &browser.SeriesFilter{Groups: []browser.Group{browser.AirTemperature, browser.SoilTemperature}},
			ctx: createContext(t, browser.Public, true),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, air_t_avg FROM air_t_avg WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"measurements_fullaccess": {
			in:  &browser.SeriesFilter{Groups: []browser.Group{browser.WindSpeed, browser.SunshineDuration}, WithSTD: true},
			ctx: createContext(t, browser.FullAccess, true),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, sun_count_tot, wind_speed, wind_speed_avg, wind_speed_max, wind_speed_std FROM sun_count_tot, wind_speed, wind_speed_avg, wind_speed_max, wind_speed_std WHERE time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"station": {
			in:  &browser.SeriesFilter{Stations: []string{"1"}},
			ctx: context.Background(),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE snipeit_location_ref='1' AND time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"stations": {
			in:  &browser.SeriesFilter{Stations: []string{"s1", "s2"}},
			ctx: context.Background(),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude FROM /.*/ WHERE snipeit_location_ref='s1' OR snipeit_location_ref='s2' AND time >= '0000-12-31T23:00:00Z' AND time <= '0001-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
		"full": {
			in: &browser.SeriesFilter{
				Groups:   []browser.Group{browser.AirTemperature, browser.WindSpeed, browser.SnowHeight},
				Stations: []string{"s1", "s2"},
				Start:    time.Date(2020, 1, 1, 0, 0, 0, 0, browser.Location),
				End:      time.Date(2020, 1, 1, 0, 0, 0, 0, browser.Location),
			},
			ctx: createContext(t, browser.FullAccess, true),
			want: &browser.Stmt{
				Query:    "SELECT station, landuse, altitude as elevation, latitude, longitude, air_t_avg, snow_height, wind_speed, wind_speed_avg, wind_speed_max FROM air_t_avg, snow_height, wind_speed, wind_speed_avg, wind_speed_max WHERE snipeit_location_ref='s1' OR snipeit_location_ref='s2' AND time >= '2019-12-31T23:00:00Z' AND time <= '2020-01-01T22:59:59Z' ORDER BY time ASC TZ('Etc/GMT-1')",
				Database: dbName,
			},
		},
	}

	db, err := NewDB(&mock.InfluxClient{
		QueryFn: queryFnTestHelper(t, ""),
	}, dbName)
	if err != nil {
		t.Fatalf("TestQuery: error in NewDB: %v", err)
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := db.Query(tc.ctx, tc.in)

			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSeries(t *testing.T) {

	// In tests we use always the same message since we use a mock implementation
	// of the influx client interface which simple returns a client.Response from
	// a give JSON file.
	testMessage := &browser.SeriesFilter{
		Groups:   []browser.Group{browser.AirTemperature, browser.RelativeHumidity, browser.SnowHeight},
		Stations: []string{"39", "4"},
		Start:    time.Date(2020, 5, 4, 0, 0, 0, 0, browser.Location),
		End:      time.Date(2020, 5, 4, 0, 0, 0, 0, browser.Location),
	}

	testCases := map[string]struct {
		in      *browser.SeriesFilter
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
			queryFn: queryFnTestHelper(t, "missing.json"),
			want: browser.TimeSeries{
				&browser.Measurement{
					Label: "air_rh_avg",
					Station: &browser.Station{
						Name:      "b1",
						Landuse:   "me",
						Elevation: 990,
						Latitude:  46.6612188656,
						Longitude: 10.5902491243,
					},
					Aggregation: "avg",
					Unit:        "%",
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
			queryFn: queryFnTestHelper(t, "multiple.json"),
			want: browser.TimeSeries{
				&browser.Measurement{
					Label:       "air_rh_avg",
					Aggregation: "avg",
					Unit:        "%",
					Station: &browser.Station{
						Name:      "b1",
						Landuse:   "me",
						Elevation: 990,
						Latitude:  46.6612188656,
						Longitude: 10.5902491243,
					},
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", math.NaN()),
						testPoint(t, "2020-05-04T00:15:00+01:00", 48.1),
						testPoint(t, "2020-05-04T00:30:00+01:00", 45.6),
						testPoint(t, "2020-05-04T00:45:00+01:00", 46.93),
						testPoint(t, "2020-05-04T01:00:00+01:00", 48.98),
					},
				},
				&browser.Measurement{
					Label: "air_rh_avg",
					Station: &browser.Station{
						Name:      "b2",
						Landuse:   "me",
						Elevation: 1490,
						Latitude:  46.6862577024,
						Longitude: 10.5798451965,
					},
					Aggregation: "avg",
					Unit:        "%",
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
					Aggregation: "avg",
					Unit:        "deg c",
					Station: &browser.Station{
						Name:      "b1",
						Landuse:   "me",
						Elevation: 990,
						Latitude:  46.6612188656,
						Longitude: 10.5902491243,
					},
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 10.05),
						testPoint(t, "2020-05-04T00:15:00+01:00", 9.46),
						testPoint(t, "2020-05-04T00:30:00+01:00", 9.61),
						testPoint(t, "2020-05-04T00:45:00+01:00", 9.72),
						testPoint(t, "2020-05-04T01:00:00+01:00", 9.02),
					},
				},
				&browser.Measurement{
					Label: "air_t_avg",
					Station: &browser.Station{
						Name:      "b2",
						Landuse:   "me",
						Elevation: 1490,
						Latitude:  46.6862577024,
						Longitude: 10.5798451965,
					},
					Aggregation: "avg",
					Unit:        "deg c",
					Points: []*browser.Point{
						testPoint(t, "2020-05-04T00:00:00+01:00", 7.379),
						testPoint(t, "2020-05-04T00:15:00+01:00", 6.933),
						testPoint(t, "2020-05-04T00:30:00+01:00", 6.783),
						testPoint(t, "2020-05-04T00:45:00+01:00", 6.53),
					},
				},
				&browser.Measurement{
					Label:       "snow_height",
					Aggregation: "smp",
					Unit:        "",
					Station: &browser.Station{
						Name:      "b1",
						Landuse:   "me",
						Elevation: 990,
						Latitude:  46.6612188656,
						Longitude: 10.5902491243,
					},
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

	c := &mock.InfluxClient{
		QueryFn: queryFnTestHelper(t, ""),
	}
	db, err := NewDB(c, "testdb")
	if err != nil {
		t.Fatalf("NewDB returned an error: %v", err)
	}
	ctx := context.Background()

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

func TestGroupsByStation(t *testing.T) {
	db, err := NewDB(&mock.InfluxClient{
		QueryFn: queryFnTestHelper(t, ""),
	}, "test")
	if err != nil {
		t.Fatalf("TestQuery: error in NewDB: %v", err)
	}

	t.Run("notfound", func(t *testing.T) {
		ctx := context.Background()
		_, err := db.GroupsByStation(ctx, 8888)
		if err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("public", func(t *testing.T) {
		want := []browser.Group{
			browser.RelativeHumidity,
			browser.AirTemperature,
			browser.ShortWaveRadiation,
			browser.SnowHeight,
			browser.WindDirection,
			browser.WindSpeed,
		}

		// Test with user public user embedded in the ctx.
		ctx := createContext(t, browser.Public, false)
		got, err := db.GroupsByStation(ctx, 3)
		if err != nil {
			t.Fatal("got error")
		}

		diff := cmp.Diff(want, got)
		if diff != "" {
			t.Fatalf("mismatch (-want +got):\n%s", diff)
		}

		// Test with general context.
		ctx = context.Background()
		got, err = db.GroupsByStation(ctx, 3)
		if err != nil {
			t.Fatal("got error")
		}

		diff = cmp.Diff(want, got)
		if diff != "" {
			t.Fatalf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("fullaccess", func(t *testing.T) {
		want := []browser.Group{
			browser.RelativeHumidity,
			browser.AirTemperature,
			browser.NDVIRadiations,
			browser.LongWaveRadiation,
			browser.ShortWaveRadiation,
			browser.PhotosyntheticallyActiveRadiation,
			browser.PRIRadiations,
			browser.SoilHeatFlux,
			browser.SnowHeight,
			browser.SunshineDuration,
			browser.SoilDielectricPermittivity,
			browser.SoilElectricalConductivity,
			browser.SoilTemperature,
			browser.SoilWaterContent,
			browser.SoilWaterPotential,
			browser.WindDirection,
			browser.WindSpeed,
		}

		ctx := createContext(t, browser.FullAccess, true)
		got, err := db.GroupsByStation(ctx, 3)
		if err != nil {
			t.Fatal("got error")
		}

		diff := cmp.Diff(want, got)
		if diff != "" {
			t.Fatalf("mismatch (-want +got):\n%s", diff)
		}
	})
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

func queryFnTestHelper(t *testing.T, filename string) func(q client.Query) (*client.Response, error) {
	t.Helper()

	return func(q client.Query) (*client.Response, error) {
		inQuery := strings.ToLower(q.Command)

		switch {
		case strings.HasPrefix(inQuery, "show measurements"):
			filename = "measurements.json"
		case strings.HasPrefix(inQuery, "show tag"):
			filename = "tags.json"
		}

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

// createContext returns a new context with an browser.User embedded with the
// given role and license.
func createContext(t *testing.T, role browser.Role, lic bool) context.Context {
	t.Helper()

	u := &browser.User{
		Role:    role,
		License: lic,
	}
	return context.WithValue(context.Background(), browser.UserContextKey, u)
}
