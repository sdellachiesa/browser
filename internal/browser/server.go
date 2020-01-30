// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	text "text/template"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/static"
)

// Backend is an interface for retrieving LTER data.
type Backend interface {
	Get(auth.Role) Stations
	Series(context.Context, *request) ([][]string, error)
	Query(context.Context, *request) string
}

// Server represents an HTTP server for serving the LTER Browser
// application.
type Server struct {
	basePath string
	mux      *http.ServeMux

	html struct {
		index *template.Template
	}

	text struct {
		python, rlang *text.Template
	}

	// Influx database name used inside code templates.
	database string

	db Backend
}

// NewServer initializes and returns a new HTTP server. It takes
// one or more option function and applies them in order to the
// server.
func NewServer(options ...Option) (*Server, error) {
	s := &Server{
		basePath: "static",
		mux:      http.NewServeMux(),
	}

	for _, option := range options {
		option(s)
	}

	if err := s.parseTemplate(); err != nil {
		return nil, fmt.Errorf("parsing templates: %v", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("must provide and option func that specifies a Backend")
	}

	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/static/", static.ServeContent(".tmpl", ".html"))

	s.mux.HandleFunc("/api/v1/series", s.handleSeries)
	s.mux.HandleFunc("/api/v1/template", grantAccessTo(s.handleTemplate, auth.FullAccess))

	return s, nil
}

// Option controls some aspects of the server.
type Option func(*Server)

// WithBackend returns an option function for setting
// the server's backend.
func WithBackend(b Backend) Option {
	return func(s *Server) {
		s.db = b
	}
}

// WithDatabase returns an options function for setting
// the server's database used insde the code templates.
func WithInfluxDB(db string) Option {
	return func(s *Server) {
		s.database = db
	}
}

func (s *Server) parseTemplate() error {
	index, err := static.File(filepath.Join(s.basePath, "index.html"))
	if err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"Join":        strings.Join,
		"Landuse":     MapLanduse,
		"Measurement": MapMeasurements,
	}
	s.html.index, err = template.New("base").Funcs(funcMap).Parse(index)
	if err != nil {
		return err
	}

	python, err := static.File(filepath.Join(s.basePath, "templates", "python.tmpl"))
	if err != nil {
		return err
	}
	s.text.python, err = text.New("python").Parse(python)

	rlang, err := static.File(filepath.Join(s.basePath, "templates", "r.tmpl"))
	if err != nil {
		return err
	}
	s.text.rlang, err = text.New("r").Parse(rlang)

	return err
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	err := s.html.index.Execute(w, struct {
		Data            Stations
		StartDate       string
		EndDate         string
		IsAuthenticated bool
		Role            auth.Role
	}{
		s.db.Get(role),
		time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
		auth.IsAuthenticated(r),
		role,
	})
	if err != nil {
		err = fmt.Errorf("handleIndex: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

// request represents an request received from the client.
type request struct {
	measurements, stations, landuse []string
	start                           time.Time
	end                             time.Time
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	req, err := parseForm(r)
	if err != nil {
		err = fmt.Errorf("handleSeries: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	ctx := r.Context()
	b, err := s.db.Series(ctx, req)
	if err != nil {
		err = fmt.Errorf("handleSeries: %v", err)
		reportError(w, r, err)
		return
	}

	filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.csv", time.Now().Unix())
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	csv.NewWriter(w).WriteAll(b)
}

func (s *Server) handleTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	req, err := parseForm(r)
	if err != nil {
		err = fmt.Errorf("handleTemplate: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	var (
		tmpl *text.Template
		ext  string
	)
	switch r.FormValue("language") {
	case "python":
		tmpl = s.text.python
		ext = "py"
	case "r":
		tmpl = s.text.rlang
		ext = "r"
	default:
		reportError(w, r, errors.New("language not supported"))
		return
	}

	filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.%s", time.Now().Unix(), ext)
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	ctx := r.Context()
	q := s.db.Query(ctx, req)

	err = tmpl.Execute(w, struct {
		Query    string
		Database string
	}{
		Query:    q,
		Database: s.database,
	})
	if err != nil {
		err = fmt.Errorf("handleTemplate: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set default security headers.
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "deny")

	s.mux.ServeHTTP(w, r)
}

// parseForm parses form values from the given http.Request and returns
// an request. It performs basic validation for end date and download
// limit.  Data in influx is UTC but LTER data is UTC+1 therefor
// parseForm will adapt start and end times. It will shift the start
// time to -1 hour and will set the end time to 22:59:59 in order to
// capture a full day.
func parseForm(r *http.Request) (*request, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	start, err := time.Parse("2006-01-02", r.FormValue("startDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.Parse("2006-01-02", r.FormValue("endDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	// Limit download of data to one year
	limit := time.Date(end.Year()-1, end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if start.Before(limit) {
		return nil, errors.New("time range is greater then a year")
	}

	if r.Form["measurements"] == nil {
		return nil, errors.New("at least one measurement must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("at least one station must be given")
	}

	return &request{
		measurements: r.Form["measurements"],
		stations:     r.Form["stations"],
		landuse:      r.Form["landuse"],
		start:        start.Add(-1 * time.Hour),
		end:          time.Date(end.Year(), end.Month(), end.Day(), 22, 59, 59, 59, time.UTC),
	}, nil
}

func grantAccessTo(h http.HandlerFunc, roles ...auth.Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowed(r, roles...) {
			http.NotFound(w, r)
			return
		}

		h(w, r)
	}
}

func isAllowed(r *http.Request, roles ...auth.Role) bool {
	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		return false
	}

	for _, v := range roles {
		if role == v {
			return true
		}
	}

	return false
}

func reportError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%v\n", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
