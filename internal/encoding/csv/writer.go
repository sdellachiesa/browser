// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package csv writes comma-separated values (CSV) files using the LTER default
// CSV format.
//
// The format looks as follows:
//
//  time,station,landuse,elevation,latitude,longitude,a_avg,wind_speed,air_rh_avg,precip_rt_nrt_tot
//  ,,,,,,c,km/h,%,mm
//  2020-01-01 00:15:00,s1,me_s1,1000,3.14159,2.71828,0,0,0,0
//  2020-01-01 00:30:00,s1,me_s1,1000,3.14159,2.71828,1,1,1,1
//  2020-01-01 00:45:00,s1,me_s1,1000,3.14159,2.71828,2,2,2,2
//  2020-01-01 00:15:00,s2,me_s2,1000,3.14159,2.71828,0,0,0,0
//  2020-01-01 00:30:00,s2,me_s2,1000,3.14159,2.71828,1,1,1,1
//  2020-01-01 00:45:00,s2,me_s2,1000,3.14159,2.71828,2,2,2,2
//
package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/euracresearch/browser"
)

// DefaultTimeFormat defines the default format for timestamps in the CSV
// output.
const DefaultTimeFormat = "2006-01-02 15:04:05"

// Writer writes a browser.TimeSeries as a CSV file. It wraps a default
// csv.Writer.
type Writer struct {
	w *csv.Writer

	// rows represent a buffer for holding individual rows of the CSV file.
	rows [][]string

	// pos records the column position of a measurement and ensures that the
	// measurement is written only once to the header.
	pos map[string]int
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   csv.NewWriter(w),
		pos: make(map[string]int),
	}
}

type stationRange struct {
	start, end int
}

// Write writes the given browser.TimeSeries as CSV file.
func (w *Writer) Write(ts browser.TimeSeries) error {
	if len(ts) == 0 {
		return browser.ErrDataNotFound
	}
	// Sort timeseries by station.
	sort.Slice(ts, func(i, j int) bool { return ts[i].Station.Name < ts[j].Station.Name })

	w.writeHeaderAndUnits(ts)

	// stationPosMap is map which stores the starting and ending line number of
	// a station in the row buffer.
	stationPosMap := make(map[string]*stationRange)

	for _, m := range ts {
		// Sort points by timestamp.
		sort.Slice(m.Points, func(i, j int) bool { return m.Points[i].Timestamp.Before(m.Points[j].Timestamp) })

		row, ok := stationPosMap[m.Station.Name]
		if !ok {
			// Station is not present in the row buffer. For each point append a
			// new line to the buffer.
			for i, p := range m.Points {
				w.rows = append(w.rows, w.newLine(m, p))

				// Store the staring row number of the current station on the
				// first processed point.
				if i == 0 {
					stationPosMap[m.Station.Name] = &stationRange{start: len(w.rows) - 1}
				}

				stationPosMap[m.Station.Name].end = len(w.rows)
			}
			continue
		}

		// Station is already present in the row buffer.
		for i, p := range m.Points {
			current := row.start + i

			// If measurements of the same station have differenet lengths of
			// points, it can happen that we overflow the current row buffer so
			// a newline must be added rather than appending only the value to a
			// existing one.
			if len(w.rows) <= current {
				w.rows = append(w.rows, w.newLine(m, p))
				stationPosMap[m.Station.Name].end = len(w.rows)
				continue
			}

			// Scan each row of the current station and check where to insert or
			// append the point according to its timestamp.
			for j := current; j <= row.end; j++ {
				t, err := time.ParseInLocation(DefaultTimeFormat, w.rows[j][0], browser.Location)
				if err != nil {
					continue
				}

				// If the current timestamp of the point is before the current
				// lines timestamp add it at the current position and shift all
				// lines by one. Timestamps of the points are always sorted.
				if p.Timestamp.Before(t) {
					// insert a row at the given current row number.
					// https://github.com/golang/go/wiki/SliceTricks#insert
					w.rows = append(w.rows, []string{})
					copy(w.rows[j+1:], w.rows[j:])
					w.rows[j] = w.newLine(m, p)
					break
				}

				if p.Timestamp.Equal(t) {
					column, ok := w.pos[m.Label]
					if !ok {
						break
					}
					w.rows[j][column] = fmt.Sprint(p.Value)
					break
				}
			}
		}
	}

	return w.w.WriteAll(w.rows)
}

// newLine creates a new line from the given browser.Measurement.
func (w *Writer) newLine(m *browser.Measurement, p *browser.Point) []string {
	length := w.rows[0]

	line := make([]string, len(length))
	// fill line with NaN's
	for i := 0; i < len(length); i++ {
		line[i] = "NaN"
	}

	line[0] = p.Timestamp.Format(DefaultTimeFormat)
	line[1] = m.Station.Name
	line[2] = m.Station.Landuse
	line[3] = fmt.Sprint(m.Station.Elevation)
	line[4] = fmt.Sprint(m.Station.Latitude)
	line[5] = fmt.Sprint(m.Station.Longitude)

	pos, ok := w.pos[m.Label]
	if ok {
		line[pos] = fmt.Sprint(p.Value)
	}

	return line
}

// writeHeaderAndUnits writes the header and unit rows to the line buffer.
func (w *Writer) writeHeaderAndUnits(ts browser.TimeSeries) {
	// Write header and empty unit line.
	w.rows = append(w.rows, []string{"time", "station", "landuse", "elevation", "latitude", "longitude"})
	w.rows = append(w.rows, []string{"", "", "", "", "", ""})

	for _, m := range ts {
		_, ok := w.pos[m.Label]
		if !ok {
			// Label is not present in the header so we will add it and store
			// its column position.
			w.appendToLine(0, m.Label)
			w.pos[m.Label] = len(w.rows[0]) - 1

			// Write unit below label.
			w.appendToLine(1, m.Unit)
		}
	}
}

// appendToLine appens the given content to the end of the given row number. If
// the give number is out of range a new line will be added.
func (w *Writer) appendToLine(row int, content string) {
	// Check if n is out of range. If so create a new line instead of appending
	// to an existing one.
	if row >= len(w.rows) {
		w.rows = append(w.rows, []string{content})
		return
	}

	w.rows[row] = append(w.rows[row], content)
}
