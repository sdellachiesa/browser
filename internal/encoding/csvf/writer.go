// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package csvf writes comma-separated values (CSV) files using the LTER
// friendly format.
//
// The friendly format has the header vertical and values in horizontal order.
// Here is an example of the friendly CSV output:
//
//      station,b1,b1,b1,b2,b2
//      landuse,me,me,me,me,me
//      latitude,46.6612188656,46.6612188656,46.6612188656,46.6862577024,46.6862577024
//      longitude,10.5902491243,10.5902491243,10.5902491243,10.5798451965,10.5798451965
//      elevation,990,990,990,1490,1490
//      parameter,precip_rt_nrt,snow_height,wind_dir,snow_height,wind_dir
//      depth,,,,,
//      aggregation,tot,smp,smp,smp,smp
//      unit,mm,,degrees,,degrees
//      2020-01-07 00:00:00,0,0.028,77,0.122,42
//      2020-01-07 00:15:00,0,0.027,115,0.128,83
//      2020-01-07 00:30:00,0,0.03,69,0.128,36
//      ...
//
// For more information see:
// https://github.com/euracresearch/browser/-/issues/90
package csvf

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/euracresearch/browser"
)

// DefaultTimeFormat defines the default format to timestamp in the CSV output.
const DefaultTimeFormat = "2006-01-02 15:04:05"

// Writer writes a browser.TimeSeries as a friendly CSV file. It wraps a default
// csv.Writer.
type Writer struct {
	w *csv.Writer

	// rows is used as a buffer holding all rows for appending values.
	rows [][]string
}

// NewWriter returns a new Writer that writes too w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: csv.NewWriter(w),
	}
}

// Write writes the given browser.TimeSeries as friendly CSV file.
func (w *Writer) Write(ts browser.TimeSeries) error {
	if len(ts) == 0 {
		return browser.ErrDataNotFound
	}

	// Sort time series by station.
	sort.Slice(ts, func(i, j int) bool { return ts[i].Station.Name < ts[j].Station.Name })

	w.writeHeader("station", "landuse", "latitude", "longitude", "elevation", "parameter", "depth", "aggregation", "unit")

	// maxColumns is the length of the time series plus the header.
	maxColumns := len(ts) + 1
	for k, m := range ts {
		w.appendToRow(0, m.Station.Name)
		w.appendToRow(1, m.Station.Landuse)
		w.appendToRow(2, fmt.Sprint(m.Station.Latitude))
		w.appendToRow(3, fmt.Sprint(m.Station.Longitude))
		w.appendToRow(4, fmt.Sprint(m.Station.Elevation))
		w.appendToRow(5, name(m))
		w.appendToRow(6, depth(m.Depth))
		w.appendToRow(7, m.Aggregation)
		w.appendToRow(8, m.Unit)

		// Sort points by timestamp.
		sort.Slice(m.Points, func(i, j int) bool { return m.Points[i].Timestamp.Before(m.Points[j].Timestamp) })

		for i, p := range m.Points {
			current := 9 + i

			// For the first measurement or if the current measurement has more
			// points than previous ones, create a new row and write the
			// timestamp and the value at the specific column.
			if k == 0 || len(w.rows) <= current {
				row := make([]string, maxColumns)
				for j := 0; j < maxColumns; j++ {
					row[j] = "NaN"
				}

				row[0] = p.Timestamp.Format(DefaultTimeFormat)
				row[k+1] = fmt.Sprint(p.Value)
				w.appendRow(row)
				continue
			}

			t, err := time.ParseInLocation(DefaultTimeFormat, w.rows[current][0], browser.Location)
			if err != nil {
				return err
			}

			// Check if the timestamp of the current row is equal to the
			// timestamp of the point. If not means that the measurements do not
			// have a continuous time range. This is currently not supported and
			// will through an error.
			// TODO: add support for non continuous time ranges.
			if !p.Timestamp.Equal(t) {
				return errors.New("not continuous timerange")
			}

			// Add value to the current row at the given column.
			w.rows[current][k+1] = fmt.Sprint(p.Value)
		}
	}

	return w.w.WriteAll(w.rows)
}

// writeHeader writes the given names in vertical order, line by line.
func (w *Writer) writeHeader(names ...string) {
	for _, n := range names {
		w.appendRow([]string{n})
	}
}

// appendRow appends the given line at the end of all buffered rows.
func (w *Writer) appendRow(line []string) {
	w.rows = append(w.rows, line)
}

// appendToRow appends the given data to the end of the given row. If the given
// row number is out of range a new row will be added.
func (w *Writer) appendToRow(row int, data string) {
	// Check if row is out of range. If so create a new row instead of appending
	// to an existing one.
	if row >= len(w.rows) {
		w.appendRow([]string{data})
		return
	}

	w.rows[row] = append(w.rows[row], data)
}

// name removes the depth and aggregation from the raw label.
func name(m *browser.Measurement) string {
	// Remove depth from the label if the measurement has a depth.
	if m.Depth > 0 {
		return strings.ReplaceAll(m.Label, fmt.Sprintf("_%02d_%s", m.Depth, m.Aggregation), "")
	}
	return strings.ReplaceAll(m.Label, "_"+m.Aggregation, "")
}

// depth will return the depth as string.
func depth(d int64) string {
	if d == 0 {
		return ""
	}

	return strconv.FormatInt(d, 10)
}
