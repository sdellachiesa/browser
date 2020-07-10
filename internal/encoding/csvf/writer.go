// Copyright 2020 Eurac Research. All rights reserved.

// Package csvf writes comma-separated values (CSV) files using the LTER
// friendly format.
//
// The friendly format has the header vertical and values in horizontal order.
// Here is an example of the friendly CSV output:
//
//		station,b1,b1,b1,b2,b2
//		landuse,me,me,me,me,me
//		latitude,46.6612188656,46.6612188656,46.6612188656,46.6862577024,46.6862577024
//		longitude,10.5902491243,10.5902491243,10.5902491243,10.5798451965,10.5798451965
//		elevation,990,990,990,1490,1490
//		aggregation,tot,smp,smp,smp,smp
//		unit,mm,,degrees,,degrees
//
//		time,precip_rt_nrt,snow_height,wind_dir,snow_height,wind_dir
//		2020-01-07 00:00:00,0,0.028,77,0.122,42
//		2020-01-07 00:15:00,0,0.027,115,0.128,83
//		2020-01-07 00:30:00,0,0.03,69,0.128,36
//		...
//
// For more information see:
// https://gitlab.inf.unibz.it/lter/browser/-/issues/90
package csvf

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
)

// DefaultTimeFormat defines the default format to timestamp in the CSV output.
const DefaultTimeFormat = "2006-01-02 15:04:05"

// Writer writes a browser.TimeSereis as a friendly CSV file. It wrapps a default
// csv.Writer.
type Writer struct {
	w *csv.Writer

	// rows is used as a buffer holding all rows for appending values.
	rows [][]string
}

// NewWriter returns a new Writer that writes to w.
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

	// Sort timeseries by station.
	sort.Slice(ts, func(i, j int) bool { return ts[i].Station < ts[j].Station })

	w.writeHeader("station", "landuse", "latitude", "longitude", "elevation", "aggregation", "unit", "", "time")

	var maxRowLen int
	for k, m := range ts {
		w.appendToRow(0, m.Station)
		w.appendToRow(1, m.Landuse)
		w.appendToRow(2, fmt.Sprint(m.Latitude))
		w.appendToRow(3, fmt.Sprint(m.Longitude))
		w.appendToRow(4, fmt.Sprint(m.Elevation))
		w.appendToRow(5, m.Aggregation)
		w.appendToRow(6, m.Unit)
		w.appendToRow(8, m.Name())

		// maxRowLen is always the length of the measurement name line.
		maxRowLen = len(w.rows[8])

		// Sort points by timestamp.
		sort.Slice(m.Points, func(i, j int) bool { return m.Points[i].Timestamp.Before(m.Points[j].Timestamp) })

		for i, p := range m.Points {
			current := 9 + i

			// For the first measurement write timestamp and column.
			if k == 0 {
				w.appendToRow(current, p.Timestamp.Format(DefaultTimeFormat))
				w.appendToRow(current, fmt.Sprint(p.Value))
				continue
			}

			// Check the current rows timestamp if it is equal to the point
			// timestamp. If not, means that the measurements in the timeseries
			// do not have a continouse timerange. This is currently considers
			// as a problem.
			// TODO: Try to make it handle non continuouse timeranges.

			t, err := time.ParseInLocation(DefaultTimeFormat, w.rows[current][0], browser.Location)
			if err != nil {
				return err
			}

			if !p.Timestamp.Equal(t) {
				return errors.New("not continouse timerange")
			}

			w.appendToRow(current, fmt.Sprint(p.Value))

			// TODO: This implementation of supporting non continuous timeranges is slow.
			// For all other measurements check if the timestamp is already
			// present and if so append the value to the current row. Otherwise
			// add the timestamp and shift all by one row.
			//for j := 9; j < len(w.rows); j++ {
			//				t, err := time.ParseInLocation(DefaultTimeFormat, w.rows[j][0], browser.Location)
			//				if err != nil {
			//					continue
			//				}
			//
			//				if p.Timestamp.Before(t) {
			//					w.rows = append(w.rows, []string{})
			//					copy(w.rows[j+1:], w.rows[j:])
			//					w.rows[j] = newRow(maxRowLen, p)
			//					break
			//				}
			//
			//				if p.Timestamp.Equal(t) {
			//					w.appendToRow(j, fmt.Sprint(p.Value))
			//					break
			//				}
			//			}

		}
	}

	// Check if all rows have the same length. If not expand the row to the
	// maxRowLen and fill it with NaN.
	for i := 9; i < len(w.rows); i++ {
		l := w.rows[i]
		if len(l) != maxRowLen {
			for j := len(l); j < maxRowLen; j++ {
				w.rows[i] = append(w.rows[i], "NaN")
			}
		}
	}

	return w.w.WriteAll(w.rows)
}

// newRow adds a new row with the given length and given point.
//func newRow(length int, p *browser.Point) []string {
//	r := make([]string, length)
//
//	for i := 0; i < length; i++ {
//		r[i] = "NaN"
//	}
//
//	r[0] = p.Timestamp.Format(DefaultTimeFormat)
//	r[length-1] = fmt.Sprint(p.Value)
//
//	return r
//}

// writeHeader writes the given names in vertical order, line by line.
func (w *Writer) writeHeader(names ...string) {
	for _, n := range names {
		w.appendRow([]string{n})
	}
}

// appendRow appens the given line at the end of all buffered rows.
func (w *Writer) appendRow(line []string) {
	w.rows = append(w.rows, line)
}

// appendToRow appens the given content to the end of the given number. If
// the give number is out of range a new row will be added.
func (w *Writer) appendToRow(n int, content string) {
	// Check if n is out of range. If so create a new row instead of appending
	// to an exisiting one.
	if n >= len(w.rows) {
		w.appendRow([]string{content})
		return
	}

	w.rows[n] = append(w.rows[n], content)
}
