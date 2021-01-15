// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package csv

import (
	"strings"
	"testing"
	"time"

	"github.com/euracresearch/browser"

	"github.com/google/go-cmp/cmp"
)

func TestWrite(t *testing.T) {
	testCases := map[string]struct {
		in   browser.TimeSeries
		want string
	}{
		"empty": {
			browser.TimeSeries{},
			"",
		},
		"one_station_one_measure": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 5),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg
,,,,,,c
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3
2020-01-01 01:15:00,s1,me_s1,1000,3.14159,2.71828,4
`,
		},
		"one_station_more_measurements": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 3),
				testMeasurement("wind_speed", "s1", "km/h", 3),
				testMeasurement("air_rh_avg", "s1", "%", 3),
				testMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
,,,,,,c,km/h,%,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2,2,2
`,
		},
		"two_station_same_measurement": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 5),
				testMeasurement("a_avg", "s2", "c", 5),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg
,,,,,,c
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3
2020-01-01 01:15:00,s1,me_s1,1000,3.14159,2.71828,4
2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0
2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1
2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2
2020-01-01 01:00:00,s2,me_s2,1000,3.14159,2.71828,3
2020-01-01 01:15:00,s2,me_s2,1000,3.14159,2.71828,4
`,
		},
		"two_station_more_measurements": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 3),
				testMeasurement("wind_speed", "s1", "km/h", 3),
				testMeasurement("air_rh_avg", "s1", "%", 3),
				testMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
				testMeasurement("a_avg", "s2", "c", 3),
				testMeasurement("wind_speed", "s2", "km/h", 3),
				testMeasurement("air_rh_avg", "s2", "%", 3),
				testMeasurement("precip_rt_nrt_tot", "s2", "mm", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
,,,,,,c,km/h,%,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2,2,2
2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2,2,2,2
`,
		},
		"two_station_three_measurements": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 3),
				testMeasurement("a_avg", "s2", "c", 3),
				testMeasurement("air_rh_avg", "s2", "mm", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,air_rh_avg
,,,,,,c,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,NaN
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,NaN
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,NaN
2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0,0
2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1,1
2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2,2
`,
		},
		"three_station_more_measurements_with_missing": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 3),
				testMeasurement("wind_speed", "s1", "km/h", 3),
				testMeasurement("air_rh_avg", "s1", "%", 3),
				testMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
				testMeasurement("a_avg", "s2", "c", 3),
				testMeasurement("wind_speed", "s2", "km/h", 3),
				testMeasurement("air_rh_avg", "s3", "%", 3),
				testMeasurement("precip_rt_nrt_tot", "s2", "mm", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
,,,,,,c,km/h,%,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2,2,2
2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0,0,NaN,0
2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1,1,NaN,1
2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2,2,NaN,2
2020-01-01 00:15:00,s3,me_s3,1000,3.14159,2.71828,NaN,NaN,0,NaN
2020-01-01 00:30:00,s3,me_s3,1000,3.14159,2.71828,NaN,NaN,1,NaN
2020-01-01 00:45:00,s3,me_s3,1000,3.14159,2.71828,NaN,NaN,2,NaN
`,
		},
		"three_station_more_measurements_with_missing_not_equal": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 2),
				testMeasurement("a_avg", "s2", "c", 3),
				testMeasurement("air_rh_avg", "s1", "%", 3),
				testMeasurement("wind_speed", "s2", "km/h", 3),
				testMeasurement("air_rh_avg", "s3", "%", 1),
				testMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
				testMeasurement("precip_rt_nrt_tot", "s2", "mm", 3),
				testMeasurement("wind_speed", "s1", "km/h", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
,,,,,,c,km/h,%,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,NaN,2,2,2
2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0,0,NaN,0
2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1,1,NaN,1
2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2,2,NaN,2
2020-01-01 00:15:00,s3,me_s3,1000,3.14159,2.71828,NaN,NaN,0,NaN
`,
		},
		"not_continuous_time_between_measurements": {
			browser.TimeSeries{
				&browser.Measurement{
					Label: "a_avg",
					Unit:  "c",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T01:00:00+01:00", 3),
					},
				},
				&browser.Measurement{
					Label: "b_avg",
					Unit:  "mm",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:30:00+01:00", 1),
					},
				},
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,b_avg
,,,,,,c,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,NaN,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3,NaN
`,
		},
		"one_station_two_measurements_different_starttime_not_sorted": {
			browser.TimeSeries{
				&browser.Measurement{
					Label: "a_avg",
					Unit:  "c",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T01:00:00+01:00", 3),
					},
				},
				&browser.Measurement{
					Label: "c_avg",
					Unit:  "mm",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:00:00+01:00", 0),
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:30:00+01:00", 6),
					},
				},
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,c_avg
,,,,,,c,mm
2020-01-01 00:00:00,s1,me_s1,1000,3.14159,2.71828,NaN,0
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,NaN
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,NaN,6
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3,NaN
`,
		},
		"one_station_more_measurements_different_time_intervals_not_sorted": {
			browser.TimeSeries{
				&browser.Measurement{
					Label: "a_avg",
					Unit:  "c",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T01:00:00+01:00", 3),
					},
				},
				&browser.Measurement{
					Label: "c_avg",
					Unit:  "mm",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:02:00+01:00", 0),
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:46:00+01:00", 6),
					},
				},
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,c_avg
,,,,,,c,mm
2020-01-01 00:02:00,s1,me_s1,1000,3.14159,2.71828,NaN,0
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,NaN
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2
2020-01-01 00:46:00,s1,me_s1,1000,3.14159,2.71828,NaN,6
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3,NaN
`,
		},
		"more_station_more_measurements_different_time_intervals_not_sorted": {
			browser.TimeSeries{
				&browser.Measurement{
					Label: "a_avg",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Unit: "c",
					Points: []*browser.Point{
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T01:00:00+01:00", 3),
					},
				},
				&browser.Measurement{
					Label: "c_avg",
					Unit:  "mm",
					Station: &browser.Station{
						Name:      "s1",
						Landuse:   "me_s1",
						Elevation: 1000,
						Latitude:  3.14159,
						Longitude: 2.71828,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:02:00+01:00", 0),
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:46:00+01:00", 6),
					},
				},
				&browser.Measurement{
					Label: "c_avg",
					Unit:  "mm",
					Station: &browser.Station{
						Name:      "s2",
						Landuse:   "me_s2",
						Elevation: 50,
						Latitude:  3,
						Longitude: 2,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:30:00+01:00", 10),
						testPoint("2020-01-01T00:45:00+01:00", 22),
						testPoint("2020-01-01T01:00:00+01:00", 66),
					},
				},
				&browser.Measurement{
					Label: "x_avg",
					Unit:  "cm",
					Station: &browser.Station{
						Name:      "s0",
						Landuse:   "me_s0",
						Elevation: 900,
						Latitude:  3.141,
						Longitude: 2.71,
					},
					Points: []*browser.Point{
						testPoint("2020-01-01T00:45:00+01:00", 2),
						testPoint("2020-01-01T00:15:00+01:00", 0),
						testPoint("2020-01-01T01:00:00+01:00", 3),
					},
				},
			},
			`time,station,landuse,elevation,latitude,longitude,x_avg,a_avg,c_avg
,,,,,,cm,c,mm
2020-01-01 00:15:00,s0,me_s0,900,3.141,2.71,0,NaN,NaN
2020-01-01 00:45:00,s0,me_s0,900,3.141,2.71,2,NaN,NaN
2020-01-01 01:00:00,s0,me_s0,900,3.141,2.71,3,NaN,NaN
2020-01-01 00:02:00,s1,me_s1,1000,3.14159,2.71828,NaN,NaN,0
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,NaN,0,NaN
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,NaN,2,2
2020-01-01 00:46:00,s1,me_s1,1000,3.14159,2.71828,NaN,NaN,6
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,NaN,3,NaN
2020-01-01 00:30:00,s2,me_s2,50,3,2,NaN,NaN,10
2020-01-01 00:45:00,s2,me_s2,50,3,2,NaN,NaN,22
2020-01-01 01:00:00,s2,me_s2,50,3,2,NaN,NaN,66
`,
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			var buf strings.Builder
			w := NewWriter(&buf)
			w.Write(tc.in)

			diff := cmp.Diff(tc.want, buf.String())
			if diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func testMeasurement(label, station, unit string, n int) *browser.Measurement {
	m := &browser.Measurement{
		Label: label,
		Unit:  unit,
		Station: &browser.Station{
			Name:      station,
			Landuse:   "me_" + station,
			Elevation: 1000,
			Latitude:  3.14159,
			Longitude: 2.71828,
		},
	}

	ts := time.Date(2020, time.January, 1, 0, 0, 0, 0, browser.Location)

	for i := 0; i < n; i++ {
		ts = ts.Add(15 * time.Minute)
		m.Points = append(m.Points, &browser.Point{
			Timestamp: ts,
			Value:     float64(i),
		})
	}

	return m
}

func testPoint(t string, value float64) *browser.Point {
	ts, _ := time.ParseInLocation(time.RFC3339, t, browser.Location)
	return &browser.Point{
		Timestamp: ts,
		Value:     value,
	}
}
