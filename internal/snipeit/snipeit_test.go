// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package snipeit

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/euracresearch/browser"
	"github.com/google/go-cmp/cmp"
)

var (
	mux        *http.ServeMux // mux is the HTTP request multiplexer used with the test server.
	testClient *StationService
)

func TestStation(t *testing.T) {
	ctx := context.Background()

	mux.HandleFunc("/locations/", func(w http.ResponseWriter, r *http.Request) {
		id := path.Base(r.URL.Path)

		switch id {
		default:
			http.NotFound(w, r)
			return

		case "2":
			b, err := ioutil.ReadFile("testdata/single.json")
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			w.Write(b)
		case "4":
			b, err := ioutil.ReadFile("testdata/single_parse_error.json")
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			w.Write(b)
		}
	})

	t.Run("OK", func(t *testing.T) {
		got, err := testClient.Station(ctx, 2)
		if err != nil {
			t.Fatalf("Station returned error: %v", err)
		}

		want := &browser.Station{
			ID:        2,
			Name:      "T1",
			Landuse:   "pa",
			Elevation: 1526,
			Latitude:  46.685863,
			Longitude: 10.58294569,
			Image:     "T1.jpg",
			Dashboard: "http://grafana/T1",
		}

		diff := cmp.Diff(want, got)
		if diff != "" {
			t.Fatalf("mismatch (-want +got):\n%s", diff)
		}

	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := testClient.Station(ctx, 3)
		if err == nil {
			t.Fatalf(": %v", err)
		}
	})

	t.Run("ParseError", func(t *testing.T) {
		_, err := testClient.Station(ctx, 4)
		if err == nil {
			t.Fatalf(": %v", err)
		}
	})

}

func TestStations(t *testing.T) {
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadFile("testdata/multiple.json")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})
	ctx := context.Background()
	t.Run("Ok", func(t *testing.T) {
		stations, err := testClient.Stations(ctx)
		if err != nil {
			t.Fatalf("Stations returned error: %v", err)
		}

		want := 3
		if got := len(stations); got != want {
			t.Fatalf("mismatch want %d, got %d", want, got)
		}
	})
}

func TestMain(m *testing.M) {
	mux = http.NewServeMux()

	// Run Mock SnipeIT API
	server := httptest.NewServer(mux)

	var err error
	testClient, err = NewStationService(server.URL, "testtoken")
	if err != nil {
		log.Fatalf("NewSnipeITService failed: %v", err)
	}

	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}
