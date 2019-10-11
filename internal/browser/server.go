// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"
	"gitlab.inf.unibz.it/lter/browser/static"
)

// Decoder is an interface for decoding data.
type Decoder interface {
	// DecodeAndValidate decodes data from the given HTTP request and
	// validates it and returns an ql.Querier.
	DecodeAndValidate(r *http.Request) (ql.Querier, error)
}

// The Backend interface for retrieving data.
type Backend interface {
	Filter(ql.Querier) (*Filter, error)
	Series(ql.Querier) ([][]string, error)
	Stations(ids []string) ([]*Station, error)
}

// Server represents an HTTP server for serving the LTER Browser
// application.
type Server struct {
	basePath string
	mux      *http.ServeMux
	tmpl     *template.Template

	db      Backend
	decoder Decoder
}

// NewServer initializes and returns a new HTTP server. It takes
// one or more option funciton and applies them in order to the
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

	if s.decoder == nil {
		return nil, fmt.Errorf("must provide and option func that specifies a Decoder")
	}

	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/static/", static.ServeContent(".tmpl", ".html"))
	s.mux.HandleFunc("/api/v1/filter", s.handleFilter)
	s.mux.HandleFunc("/api/v1/series", s.handleSeries)

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

// WithDecoder returns an options function for setting
// the server's request decoder.
func WithDecoder(d Decoder) Option {
	return func(s *Server) {
		s.decoder = d
	}
}

func (s *Server) parseTemplate() error {
	f, err := static.File(filepath.Join(s.basePath, "index.html"))
	if err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"Landuse":     MapLanduse,
		"Measurement": MapMeasurements,
	}

	s.tmpl, err = template.New("base").Funcs(funcMap).Parse(f)
	return err
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	f, err := s.decoder.DecodeAndValidate(r)
	if err != nil {
		err = fmt.Errorf("handleIndex: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	data, err := s.db.Filter(f)
	if err != nil {
		err = fmt.Errorf("handleIndex: error in getting data from backend: %v", err)
		reportError(w, r, err)
		return
	}

	stations, err := s.db.Stations(data.Stations)
	if err != nil {
		err = fmt.Errorf("handleIndex: error in getting metadata from backend: %v", err)
		reportError(w, r, err)
		return
	}

	mapJSON, err := json.Marshal(stations)
	if err != nil {
		err = fmt.Errorf("handleIndex: error in marshaling json: %v", err)
		reportError(w, r, err)
		return
	}

	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	err = s.tmpl.Execute(w, struct {
		Stations  []*Station
		Fields    []string
		Landuse   []string
		Map       string
		StartDate string // TODO: Should also be set by the ACL/RBAC
		EndDate   string // TODO: Should also be set by the ACL/RBAC
		Role      auth.Role
	}{
		stations,
		data.Fields,
		data.Landuse,
		string(mapJSON),
		time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
		role,
	})
	if err != nil {
		err = fmt.Errorf("handleIndex: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

func (s *Server) handleFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	f, err := s.decoder.DecodeAndValidate(r)
	if err != nil {
		err = fmt.Errorf("handleFilter: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	data, err := s.db.Filter(f)
	if err != nil {
		err = fmt.Errorf("handleFilter: %v", err)
		reportError(w, r, err)
		return
	}

	b, err := json.Marshal(data)
	if err != nil {
		err = fmt.Errorf("handleFilter: %v", err)
		reportError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	f, err := s.decoder.DecodeAndValidate(r)
	if err != nil {
		err = fmt.Errorf("handleSeries: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	b, err := s.db.Series(f)
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set default security headers.
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "deny")

	s.mux.ServeHTTP(w, r)
}

func reportError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%v\n", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
