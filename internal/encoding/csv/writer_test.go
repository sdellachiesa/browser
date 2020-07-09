// Copyright 2020 Eurac Research. All rights reserved.

package csv

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gitlab.inf.unibz.it/lter/browser"
)

func TestWriter(t *testing.T) {
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
				genMeasurement("a_avg", "s1", "c", 5),
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
		"one_station_two_measure": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 5),
				genMeasurement("wind_speed", "s1", "km/h", 5),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed
,,,,,,c,km/h
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2
2020-01-01 01:00:00,s1,me_s1,1000,3.14159,2.71828,3,3
2020-01-01 01:15:00,s1,me_s1,1000,3.14159,2.71828,4,4
`,
		},
		"one_station_more_measure": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 3),
				genMeasurement("wind_speed", "s1", "km/h", 3),
				genMeasurement("air_rh_avg", "s1", "%", 3),
				genMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
			},
			`time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
,,,,,,c,km/h,%,mm
2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2,2,2
`,
		},
		"two_station_one_measure": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 5),
				genMeasurement("a_avg", "s2", "c", 5),
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
		"two_station_more_measure": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 3),
				genMeasurement("wind_speed", "s1", "km/h", 3),
				genMeasurement("air_rh_avg", "s1", "%", 3),
				genMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
				genMeasurement("a_avg", "s2", "c", 3),
				genMeasurement("wind_speed", "s2", "km/h", 3),
				genMeasurement("air_rh_avg", "s2", "%", 3),
				genMeasurement("precip_rt_nrt_tot", "s2", "mm", 3),
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
		"two_station_three_measure_one_missing": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 3),
				genMeasurement("a_avg", "s2", "c", 3),
				genMeasurement("air_rh_avg", "s2", "mm", 3),
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
		"three_station_more_measure_with_missing": {
			browser.TimeSeries{
				genMeasurement("a_avg", "s1", "c", 3),
				genMeasurement("wind_speed", "s1", "km/h", 3),
				genMeasurement("air_rh_avg", "s1", "%", 3),
				genMeasurement("precip_rt_nrt_tot", "s1", "mm", 3),
				genMeasurement("a_avg", "s2", "c", 3),
				genMeasurement("wind_speed", "s2", "km/h", 3),
				genMeasurement("air_rh_avg", "s3", "%", 3),
				genMeasurement("precip_rt_nrt_tot", "s2", "mm", 3),
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

func genMeasurement(label, station, unit string, n int) *browser.Measurement {
	m := &browser.Measurement{
		Label:     label,
		Station:   station,
		Landuse:   "me_" + station,
		Unit:      unit,
		Elevation: 1000,
		Latitude:  3.14159,
		Longitude: 2.71828,
	}

	ts := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < n; i++ {
		ts = ts.Add(15 * time.Minute)
		m.Points = append(m.Points, &browser.Point{
			Timestamp: ts,
			Value:     float64(i),
		})
	}

	return m
}
