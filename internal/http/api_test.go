// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/euracresearch/browser"
)

type testBackend struct{}

func (tb *testBackend) Series(ctx context.Context, m *browser.Message) (browser.TimeSeries, error) {
	var ts browser.TimeSeries

	measure := &browser.Measurement{
		Label:     "test",
		Station:   "station",
		Landuse:   "me",
		Unit:      "%",
		Elevation: 1000,
		Latitude:  3.14159,
		Longitude: 2.71828,
	}

	t := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		t = t.Add(15 * time.Minute)
		measure.Points = append(measure.Points, &browser.Point{
			Timestamp: t,
			Value:     float64(i),
		})
	}

	return append(ts, measure), nil
}

func (tb *testBackend) Query(ctx context.Context, m *browser.Message) *browser.Stmt {
	return &browser.Stmt{
		Database: "testdb",
		Query:    "querytestbackend",
	}
}

func TestHandleSeries(t *testing.T) {
	h := NewHandler(func(h *Handler) {
		h.db = new(testBackend)
	})

	testCases := map[string]struct {
		method          string
		statusCode      int
		respContentType string
		reqBody         string
		respBody        []byte
	}{
		"GET":                            {http.MethodGet, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"PUT":                            {http.MethodPut, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"HEAD":                           {http.MethodHead, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"PATCH":                          {http.MethodPatch, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"DELETE":                         {http.MethodDelete, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"OPTIONS":                        {http.MethodOptions, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		"Incomplete":                     {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23", nil},
		"MissingMeasurements":            {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&stations=1", nil},
		"MissingStations":                {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&measurements=a", nil},
		"MissingMeasurementsAndStations": {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&landuse=a", nil},
		"OK":                             {http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a", []byte("time,station,landuse,elevation,latitude,longitude,test\n,,,,,,%\n2020-01-01 00:15:00,station,me,1000,3.14159,2.71828,0\n2020-01-01 00:30:00,station,me,1000,3.14159,2.71828,1\n2020-01-01 00:45:00,station,me,1000,3.14159,2.71828,2\n2020-01-01 01:00:00,station,me,1000,3.14159,2.71828,3\n2020-01-01 01:15:00,station,me,1000,3.14159,2.71828,4\n")},
		"OKWithLanduse":                  {http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me", []byte("time,station,landuse,elevation,latitude,longitude,test\n,,,,,,%\n2020-01-01 00:15:00,station,me,1000,3.14159,2.71828,0\n2020-01-01 00:30:00,station,me,1000,3.14159,2.71828,1\n2020-01-01 00:45:00,station,me,1000,3.14159,2.71828,2\n2020-01-01 01:00:00,station,me,1000,3.14159,2.71828,3\n2020-01-01 01:15:00,station,me,1000,3.14159,2.71828,4\n")},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/series", strings.NewReader(tc.reqBody))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			resp := w.Result()

			if got, want := resp.StatusCode, tc.statusCode; got != want {
				t.Fatalf("got unexpected status code: %d, want %d", got, want)
			}

			if got, want := resp.Header.Get("Content-Type"), tc.respContentType; got != want {
				t.Fatalf("response header content-type: got %s, want %s", got, want)
			}

			if tc.respBody != nil {
				defer resp.Body.Close()
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("ioutil.ReadAll(resp.Body): %v", err)
				}

				if !bytes.Equal(b, tc.respBody) {
					t.Fatalf("got unexpected body: %q; want %q", b, tc.respBody)
				}
			}
		})
	}
}

func TestHandleTemplate(t *testing.T) {
	h := NewHandler(func(h *Handler) {
		h.db = new(testBackend)
	})

	tmplPython, err := template.ParseFS(templateFS, "templates/python.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	tmplRlang, err := template.ParseFS(templateFS, "templates/r.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	testCases := map[string]struct {
		method     string
		ctx        context.Context
		statusCode int
		reqBody    []byte
		tmpl       *template.Template
	}{
		"GET":             {http.MethodGet, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"PUT":             {http.MethodPut, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"HEAD":            {http.MethodHead, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"PATCH":           {http.MethodPatch, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"DELETE":          {http.MethodDelete, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"OPTIONS":         {http.MethodOptions, withCTX(browser.FullAccess), http.StatusMethodNotAllowed, nil, nil},
		"NIL":             {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, nil, nil},
		"EMPTY":           {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(``), nil},
		"Incomplete":      {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&measurements=a`), nil},
		"MissingLanguage": {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a`), nil},
		"EmtpyLanguage":   {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&language=`), nil},
		"R":               {http.MethodPost, withCTX(browser.FullAccess), http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&language=r`), tmplRlang},
		"Python":          {http.MethodPost, withCTX(browser.FullAccess), http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me&language=python`), tmplPython},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/v1/templates", bytes.NewReader(tc.reqBody))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req = req.WithContext(tc.ctx)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			resp := w.Result()

			if got, want := resp.StatusCode, tc.statusCode; got != want {
				t.Fatalf("got unexpected status code: %d, want %d", got, want)
			}

			contentType := "text/plain; charset=utf-8"
			if got, want := resp.Header.Get("Content-Type"), contentType; got != want {
				t.Fatalf("response header content-type: got %s, want %s", got, want)
			}

			if tc.tmpl != nil {
				defer resp.Body.Close()
				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("ioutil.ReadAll(resp.Body): %v", err)
				}

				var want bytes.Buffer
				err = tc.tmpl.Execute(&want, struct {
					Query    string
					Database string
				}{
					"querytestbackend", "testdb",
				})
				if err != nil {
					t.Fatalf("error executing template: %v", err)
				}

				if !bytes.Equal(b, want.Bytes()) {
					t.Fatalf("got unexpected body: %s; want %s", string(b), want.String())
				}
			}
		})
	}

}

func withCTX(role browser.Role) context.Context {
	u := &browser.User{Role: role}
	return context.WithValue(context.Background(), browser.UserContextKey, u)
}
