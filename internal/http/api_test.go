// Copyright 2020 Eurac Research. All rights reserved.

package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/static"
	"golang.org/x/net/xsrftoken"
)

// func TestHandleSeries(t *testing.T) {
// 	testCases := map[string]struct {
// 		method          string
// 		statusCode      int
// 		respContentType string
// 		reqBody         string
// 		respBody        []byte
// 	}{
// 		"MethodGet":          {http.MethodGet, http.StatusForbidden, "text/plain; charset=utf-8", "", nil},
// 		"MethodGetWithToken": {http.MethodGet, http.StatusForbidden, "text/plain; charset=utf-8", "" nil},
// 	}

// 	for k, tc := range testCases {
// 		t.Run(k, func(t *testing.T) {
// 			req := httptest.NewRequest(tc.method, "/api/v1/series", strings.NewReader(tc.reqBody))
// 			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

// 			w := httptest.NewRecorder()
// 			api.ServeHTTP(w, req)
// 			resp := w.Result()

// 			if got, want := resp.StatusCode, tc.statusCode; got != want {
// 				t.Errorf("got unexpected status code: %d, want %d", got, want)
// 			}
// 		})
// 	}
// }

type testBackend struct{}

func (tb *testBackend) SeriesV1(ctx context.Context, m *browser.Message) ([][]string, error) {
	var r [][]string
	v := []string{"test,series"}
	return append(r, v), nil
}

func (tb *testBackend) Series(ctx context.Context, m *browser.Message) (browser.TimeSeries, error) {
	return nil, errors.New("not yet implemented")
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
		h.key = "testing"
	})
	token := xsrftoken.Generate(h.key, "", "")

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
		"NoToken":                        {http.MethodPost, http.StatusForbidden, "text/plain; charset=utf-8", "", nil},
		"EmptyToken":                     {http.MethodPost, http.StatusForbidden, "text/plain; charset=utf-8", "token=bla", nil},
		"Incomplete":                     {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&token=" + token, nil},
		"MissingMeasurements":            {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&stations=1&token=" + token, nil},
		"MissingStations":                {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&measurements=a&token=" + token, nil},
		"MissingMeasurementsAndStations": {http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&landuse=a&token=" + token, nil},
		"OK":                             {http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&token=" + token, []byte("\"test,series\"\n")},
		"OKWithLanduse":                  {http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me&token=" + token, []byte("\"test,series\"\n")},
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
		h.key = "testing"
	})
	token := xsrftoken.Generate(h.key, "", "")

	tmplPython, err := static.ParseTextTemplates(nil, "python.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	tmplRlang, err := static.ParseTextTemplates(nil, "r.tmpl")
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
		"NIL":             {http.MethodPost, withCTX(browser.FullAccess), http.StatusForbidden, nil, nil},
		"EMPTY":           {http.MethodPost, withCTX(browser.FullAccess), http.StatusForbidden, []byte(``), nil},
		"EmptyToken":      {http.MethodPost, withCTX(browser.FullAccess), http.StatusForbidden, []byte(`token=`), nil},
		"WrongToken":      {http.MethodPost, withCTX(browser.FullAccess), http.StatusForbidden, []byte(`token=bla`), nil},
		"Incomplete":      {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&measurements=a&token=` + token), nil},
		"MissingLanguage": {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&token=` + token), nil},
		"EmtpyLanguage":   {http.MethodPost, withCTX(browser.FullAccess), http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&language=&token=` + token), nil},
		"R":               {http.MethodPost, withCTX(browser.FullAccess), http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&language=r&token=` + token), tmplRlang},
		"Python":          {http.MethodPost, withCTX(browser.FullAccess), http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me&language=python&token=` + token), tmplPython},
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
	return context.WithValue(context.Background(), browser.BrowserContextKey, u)
}
