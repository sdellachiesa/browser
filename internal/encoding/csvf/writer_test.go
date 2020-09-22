// Copyright 2020 Eurac Research. All rights reserved.

package csvf

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gitlab.inf.unibz.it/lter/browser"
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
		//		"not_continuous_time_between_measurements": {
		//			browser.TimeSeries{
		//				&browser.Measurement{
		//					Label:       "a_avg",
		//					Station:     "s1",
		//					Landuse:     "me_s1",
		//					Unit:        "c",
		//					Aggregation: "avg",
		//					Elevation:   1000,
		//					Latitude:    3.14159,
		//					Longitude:   2.71828,
		//					Points: []*browser.Point{
		//						testPoint("2020-01-01T00:45:00+01:00", 2),
		//						testPoint("2020-01-01T00:15:00+01:00", 0),
		//						testPoint("2020-01-01T01:00:00+01:00", 3),
		//					},
		//				},
		//				&browser.Measurement{
		//					Label:       "b_avg",
		//					Station:     "s1",
		//					Landuse:     "me_s1",
		//					Unit:        "mm",
		//					Aggregation: "avg",
		//					Elevation:   1000,
		//					Latitude:    3.14159,
		//					Longitude:   2.71828,
		//					Points: []*browser.Point{
		//						testPoint("2020-01-01T00:15:00+01:00", 0),
		//						testPoint("2020-01-01T00:45:00+01:00", 2),
		//						testPoint("2020-01-01T00:30:00+01:00", 1),
		//					},
		//				},
		//			},
		//			`station,s1,s1
		//landuse,me_s1,me_s1
		//latitude,3.14159,3.14159
		//longitude,2.71828,2.71828
		//elevation,1000,1000
		//aggregation,avg,avg
		//unit,c,mm
		//
		//time,a,b
		//2020-01-01 00:15:00,0,0
		//2020-01-01 00:30:00,NaN,1
		//2020-01-01 00:45:00,2,2
		//2020-01-01 01:00:00,3,NaN
		//`,
		//		},

		//		"one_station_two_measuremnent_with_different_timestamps": {
		//			browser.TimeSeries{
		//				&browser.Measurement{
		//					Label:       "a_avg",
		//					Station:     "s1",
		//					Landuse:     "me_s1",
		//					Unit:        "c",
		//					Aggregation: "avg",
		//					Elevation:   1000,
		//					Latitude:    3.14159,
		//					Longitude:   2.71828,
		//					Points: []*browser.Point{
		//						testPoint("2020-01-01T00:45:00+01:00", 2),
		//						testPoint("2020-01-01T00:15:00+01:00", 0),
		//						testPoint("2020-01-01T01:00:00+01:00", 3),
		//					},
		//				},
		//				&browser.Measurement{
		//					Label:       "b_avg",
		//					Station:     "s1",
		//					Landuse:     "me_s1",
		//					Unit:        "mm",
		//					Aggregation: "avg",
		//					Elevation:   1000,
		//					Latitude:    3.14159,
		//					Longitude:   2.71828,
		//					Points: []*browser.Point{
		//						testPoint("2020-01-01T00:10:00+01:00", 4),
		//						testPoint("2020-01-01T00:50:00+01:00", 5),
		//						testPoint("2020-01-01T00:30:00+01:00", 16),
		//					},
		//				},
		//			},
		//			`station,s1,s1
		//landuse,me_s1,me_s1
		//latitude,3.14159,3.14159
		//longitude,2.71828,2.71828
		//elevation,1000,1000
		//aggregation,avg,avg
		//unit,c,mm
		//
		//time,a,b
		//2020-01-01 00:10:00,NaN,4
		//2020-01-01 00:15:00,0,NaN
		//2020-01-01 00:30:00,NaN,16
		//2020-01-01 00:45:00,2,NaN
		//2020-01-01 00:50:00,NaN,5
		//2020-01-01 01:00:00,3,NaN
		//`,
		//		},

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
		Label:       label,
		Station:     station,
		Aggregation: "avg",
		Landuse:     "me_" + station,
		Unit:        unit,
		Elevation:   1000,
		Latitude:    3.14159,
		Longitude:   2.71828,
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

//func testPoint(t string, value float64) *browser.Point {
//	ts, _ := time.ParseInLocation(time.RFC3339, t, browser.Location)
//	return &browser.Point{
//		Timestamp: ts,
//		Value:     value,
//	}
//}
