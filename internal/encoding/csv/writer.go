// Copyright 2020 Eurac Research. All rights reserved.

// Package csv writes comma-separted values (CSV) files using the LTER default
// format.
package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"

	"gitlab.inf.unibz.it/lter/browser"
)

// DefaultTimeFormat defines the default format to timestamp in the CSV output.
const DefaultTimeFormat = "2006-01-02 15:04:05"

// Writer writes a browser.TimeSeries as a CSV file. It wrapps a default
// csv.Writer.
type Writer struct {
	w *csv.Writer

	// lines denotes a buffer holing each line of the CSV file.
	lines [][]string

	// pos records the column position of a measurement.  It is used to write a
	// measurement only once to the header.
	pos map[string]int
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   csv.NewWriter(w),
		pos: make(map[string]int),
	}
}

// Write writes the given browser.TimeSeries as CSV file.
func (w *Writer) Write(ts browser.TimeSeries) error {
	if len(ts) == 0 {
		return browser.ErrDataNotFound
	}

	// Sort timeseries by station.
	sort.Slice(ts, func(i, j int) bool { return ts[i].Station < ts[j].Station })

	w.writeHeaderAndUnits(ts)

	// stationPosMap is map which stores the starting line number of a station in
	// the line buffer.
	stationPosMap := make(map[string]int, len(ts))

	for _, m := range ts {
		spos, stationIsPresent := stationPosMap[m.Station]

		for i, p := range m.Points {
			if stationIsPresent {
				// The station is already present in the lines buffer so we need to
				// append the values at each line rather than adding new lines to
				// the buffer.
				pos, ok := w.pos[m.Label]
				if !ok {
					continue
				}

				// If spos+i is out of range due to the fact that one
				// measurement has less points then an other a new line must be
				// added instead of writing only the value on to an existing
				// line.
				if (spos + i) >= len(w.lines) {
					w.lines = append(w.lines, w.newLine(m, p))
				} else {
					w.lines[spos+i][pos] = fmt.Sprint(p.Value)
				}
				continue
			}

			// No station is present therefore a new line must be appended to
			// the buffered lines.
			w.lines = append(w.lines, w.newLine(m, p))

			// On the first written line record the line number. This is
			// the starting line for the station.
			if i == 0 {
				stationPosMap[m.Station] = len(w.lines) - 1
			}
		}
	}

	return w.w.WriteAll(w.lines)
}

func (w *Writer) newLine(m *browser.Measurement, p *browser.Point) []string {
	length := w.lines[0]

	line := make([]string, len(length))
	// fill line with NaN's
	for i := 0; i < len(length); i++ {
		line[i] = "NaN"
	}

	line[0] = p.Timestamp.Format(DefaultTimeFormat)
	line[1] = m.Station
	line[2] = m.Landuse
	line[3] = fmt.Sprint(m.Elevation)
	line[4] = fmt.Sprint(m.Latitude)
	line[5] = fmt.Sprint(m.Longitude)

	pos, ok := w.pos[m.Label]
	if ok {
		line[pos] = fmt.Sprint(p.Value)
	}

	return line
}

func (w *Writer) writeHeaderAndUnits(ts browser.TimeSeries) {
	// Write header and empty unit line.
	w.lines = append(w.lines, []string{"time", "station", "landuse", "elevation", "latitude", "longitude"})

	w.lines = append(w.lines, []string{"", "", "", "", "", ""})
	for _, m := range ts {
		_, ok := w.pos[m.Label]
		if !ok {
			// Label is not present in the header so we will add it and store
			// its column position.
			w.appendToLine(0, m.Label)
			w.pos[m.Label] = len(w.lines[0]) - 1

			// Write unit below label.
			w.appendToLine(1, m.Unit)
		}
	}
}

// appendToLine appens the given content to the end of the given line number. If
// the give number is out of range a new line will be added.
func (w *Writer) appendToLine(n int, content string) {
	// Check if n is out of range. If so create a new line instead of appending
	// to an exisiting one.
	if n >= len(w.lines) {
		w.lines = append(w.lines, []string{content})
		return
	}

	w.lines[n] = append(w.lines[n], content)
}
