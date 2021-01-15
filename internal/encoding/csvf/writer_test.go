// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package csvf

import (
	"bytes"
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
		"one_station_one_measurement": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 5),
			},
			`station,s1
landuse,me_s1
latitude,3.14159
longitude,2.71828
elevation,1000
parameter,a
depth,
aggregation,avg
unit,c
2020-01-01 00:15:00,0
2020-01-01 00:30:00,1
2020-01-01 00:45:00,2
2020-01-01 01:00:00,3
2020-01-01 01:15:00,4
`,
		},
		"two_station_one_measurement": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 5),
				testMeasurement("a_avg", "s2", "c", 5),
			},
			`station,s1,s2
landuse,me_s1,me_s2
latitude,3.14159,3.14159
longitude,2.71828,2.71828
elevation,1000,1000
parameter,a,a
depth,,
aggregation,avg,avg
unit,c,c
2020-01-01 00:15:00,0,0
2020-01-01 00:30:00,1,1
2020-01-01 00:45:00,2,2
2020-01-01 01:00:00,3,3
2020-01-01 01:15:00,4,4
`,
		},
		"two_with_first_less_points": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 3),
				testMeasurement("a_avg", "s2", "c", 5),
			},
			`station,s1,s2
landuse,me_s1,me_s2
latitude,3.14159,3.14159
longitude,2.71828,2.71828
elevation,1000,1000
parameter,a,a
depth,,
aggregation,avg,avg
unit,c,c
2020-01-01 00:15:00,0,0
2020-01-01 00:30:00,1,1
2020-01-01 00:45:00,2,2
2020-01-01 01:00:00,NaN,3
2020-01-01 01:15:00,NaN,4
`,
		},
		"two_with_last_less_points": {
			browser.TimeSeries{
				testMeasurement("a_avg", "s1", "c", 5),
				testMeasurement("a_avg", "s2", "c", 2),
			},
			`station,s1,s2
landuse,me_s1,me_s2
latitude,3.14159,3.14159
longitude,2.71828,2.71828
elevation,1000,1000
parameter,a,a
depth,,
aggregation,avg,avg
unit,c,c
2020-01-01 00:15:00,0,0
2020-01-01 00:30:00,1,1
2020-01-01 00:45:00,2,NaN
2020-01-01 01:00:00,3,NaN
2020-01-01 01:15:00,4,NaN
`,
		},
		"three_with_middle_less_points": {
			browser.TimeSeries{
				testMeasurement("c_avg", "s1", "c", 5),
				testMeasurement("b_avg", "s4", "b", 3),
				testMeasurement("a_avg", "s5", "a", 4),
			},
			`station,s1,s4,s5
landuse,me_s1,me_s4,me_s5
latitude,3.14159,3.14159,3.14159
longitude,2.71828,2.71828,2.71828
elevation,1000,1000,1000
parameter,c,b,a
depth,,,
aggregation,avg,avg,avg
unit,c,b,a
2020-01-01 00:15:00,0,0,0
2020-01-01 00:30:00,1,1,1
2020-01-01 00:45:00,2,2,2
2020-01-01 01:00:00,3,NaN,3
2020-01-01 01:15:00,4,NaN,NaN
`,
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			var buf bytes.Buffer
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
		Station: &browser.Station{
			Name:      station,
			Landuse:   "me_" + station,
			Elevation: 1000,
			Latitude:  3.14159,
			Longitude: 2.71828,
		},
		Aggregation: "avg",
		Unit:        unit,
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
