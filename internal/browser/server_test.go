package browser

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"time"

	"golang.org/x/net/xsrftoken"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
)

type testBackend struct{}

func (tb *testBackend) Get(r auth.Role) Stations {
	return nil
}

func (tb *testBackend) Series(ctx context.Context, req *request) ([][]string, error) {
	var r [][]string
	v := []string{"test,series"}
	return append(r, v), nil
}

func (tb *testBackend) Query(ctx context.Context, req *request) string {
	return "querytestbackend"
}

func TestHandleSeries(t *testing.T) {
	s, err := NewServer(func(s *Server) {
		s.db = &testBackend{}
		s.key = "testing"
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	token := xsrftoken.Generate(s.key, "", "/api/v1/")

	testCases := []struct {
		method          string
		statusCode      int
		respContentType string
		reqBody         string
		respBody        []byte
	}{
		{http.MethodGet, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodPut, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodHead, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodPatch, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodDelete, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodOptions, http.StatusMethodNotAllowed, "text/plain; charset=utf-8", "", nil},
		{http.MethodPost, http.StatusForbidden, "text/plain; charset=utf-8", "", nil},
		{http.MethodPost, http.StatusForbidden, "text/plain; charset=utf-8", "", nil},
		{http.MethodPost, http.StatusForbidden, "text/plain; charset=utf-8", "token=bla", nil},
		{http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&token=" + token, nil},
		{http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&stations=1&token=" + token, nil},
		{http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&measurements=a&token=" + token, nil},
		{http.MethodPost, http.StatusInternalServerError, "text/plain; charset=utf-8", "startDate=2019-07-23&endDate=2020-01-23&landuse=a&token=" + token, nil},
		{http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&token=" + token, []byte("\"test,series\"\n")},
		{http.MethodPost, http.StatusOK, "text/csv", "startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me&token=" + token, []byte("\"test,series\"\n")},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, "/api/v1/series", strings.NewReader(tc.reqBody))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		s.handleSeries(w, req)
		resp := w.Result()

		if got, want := resp.StatusCode, tc.statusCode; got != want {
			t.Errorf("got unexpected status code: %d, want %d", got, want)
		}

		if got, want := resp.Header.Get("Content-Type"), tc.respContentType; got != want {
			t.Errorf("response header content-type: got %s, want %s", got, want)
		}

		if tc.respBody != nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("ioutil.ReadAll(resp.Body): %v", err)
			}

			if !bytes.Equal(b, tc.respBody) {
				t.Errorf("got unexpected body: %q; want %q", b, tc.respBody)
			}
		}
	}
}

func TestHandleTemplate(t *testing.T) {
	s, err := NewServer(func(s *Server) {
		s.db = &testBackend{}
		s.database = "testDB"
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	testCases := []struct {
		method     string
		statusCode int
		reqBody    []byte
		tmpl       *template.Template
	}{
		{http.MethodGet, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodPut, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodHead, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodPatch, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodDelete, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodOptions, http.StatusMethodNotAllowed, nil, nil},
		{http.MethodPost, http.StatusInternalServerError, nil, nil},
		{http.MethodPost, http.StatusInternalServerError, []byte(``), nil},
		{http.MethodPost, http.StatusInternalServerError, []byte(`startDate=2019-07-23`), nil},
		{http.MethodPost, http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1`), nil},
		{http.MethodPost, http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&measurements=a`), nil},
		{http.MethodPost, http.StatusInternalServerError, []byte(`startDate=2019-07-23&endDate=2020-01-23&landuse=a`), nil},
		{http.MethodPost, http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&language=r`), s.text.rlang},
		{http.MethodPost, http.StatusOK, []byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me&language=python`), s.text.python},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, "/api/v1/template", bytes.NewReader(tc.reqBody))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()
		s.handleTemplate(w, req)
		resp := w.Result()

		if got, want := resp.StatusCode, tc.statusCode; got != want {
			t.Errorf("got unexpected status code: %d, want %d", got, want)
		}

		contentType := "text/plain; charset=utf-8"
		if got, want := resp.Header.Get("Content-Type"), contentType; got != want {
			t.Errorf("response header content-type: got %s, want %s", got, want)
		}

		if tc.tmpl != nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("ioutil.ReadAll(resp.Body): %v", err)
			}

			var want bytes.Buffer
			err = tc.tmpl.Execute(&want, struct {
				Query    string
				Database string
			}{
				"querytestbackend", s.database,
			})
			if err != nil {
				t.Fatalf("error executing template: %v", err)
			}

			if !bytes.Equal(b, want.Bytes()) {
				t.Errorf("got unexpected body: %s; want %s", string(b), want.String())
			}
		}
	}
}

func TestParseForm(t *testing.T) {
	startDate, err := time.Parse("2006-01-02", "2019-07-23")
	if err != nil {
		t.Fatalf("error parsing start date: %v", err)
	}
	startDate = startDate.Add(-1 * time.Hour)

	endDate, err := time.Parse("2006-01-02", "2020-01-23")
	if err != nil {
		t.Fatalf("error parsing start date: %v", err)
	}
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 22, 59, 59, 59, time.UTC)

	testCases := []struct {
		reqBody []byte
		want    *request
	}{
		{nil, nil},
		{[]byte(""), nil},
		{[]byte("startDate=2019-07-23"), nil},
		{[]byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1`), nil},
		{[]byte(`startDate=2019-07-23&endDate=2020-01-23&measurements=a`), nil},
		{[]byte(`startDate=2019-07-23&endDate=2020-01-23&landuse=a`), nil},
		{
			[]byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a`),
			&request{
				measurements: []string{"a"},
				stations:     []string{"1"},
				start:        startDate,
				end:          endDate,
			},
		},
		{
			[]byte(`startDate=2019-07-23&endDate=2020-01-23&stations=1&measurements=a&landuse=me`),
			&request{
				measurements: []string{"a"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
				start:        startDate,
				end:          endDate,
			},
		},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/template", bytes.NewReader(tc.reqBody))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		got, _ := parseForm(req)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got: %v; want: %v", got, tc.want)
		}
	}
}

func TestIsAllowed(t *testing.T) {
	testCases := []struct {
		in   []auth.Role
		role auth.Role
		want bool
	}{
		{[]auth.Role{}, auth.Public, false},
		{[]auth.Role{}, auth.Role("nothing"), false},
		{[]auth.Role{auth.Public}, auth.Public, true},
		{[]auth.Role{auth.Public}, auth.FullAccess, false},
		{[]auth.Role{auth.Public, auth.FullAccess}, auth.Public, true},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), auth.JWTClaimsContextKey, tc.role)
		req = req.WithContext(ctx)

		if got := isAllowed(req, tc.in...); got != tc.want {
			t.Errorf("got %v, want %v", got, tc.want)
		}
	}
}
