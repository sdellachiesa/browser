// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gitlab.inf.unibz.it/lter/browser/static"
)

// Server represents an HTTP server for serving the LTER Browser
// application.
type Server struct {
	basePath string
	db       Backend
	mux      *http.ServeMux
	tmpl     *template.Template
}

// NewServer initializes and returns a new HTTP server serving the LTER
// Browser application. It takes one or more option funciton and applies
// them in order to Server.
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
		return nil, fmt.Errorf("must provide and option func that specifies a datastore")
	}

	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/static/", s.handleStatic)
	s.mux.HandleFunc("/api/v1/filter", s.handleFilter)
	s.mux.HandleFunc("/api/v1/series", s.handleSeries)

	return s, nil
}

// Option contorls some aspects of the server.
type Option func(*Server)

// WithBackend returns an option function for setting
// the server's backend.
func WithBackend(b Backend) Option {
	return func(s *Server) {
		s.db = b
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

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[1:]

	b, err := static.File(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, path.Base(p), time.Now(), strings.NewReader(b))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	opts := &Filter{
		Fields: []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"},
	}
	f, err := s.db.Filter(opts)
	if err != nil {
		log.Printf("handleIndex: error in getting data from backend: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stations, err := s.db.Stations(f.Stations)
	if err != nil {
		log.Printf("handleIndex: error in getting metadata from backend: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapJSON, err := json.Marshal(stations)
	if err != nil {
		log.Printf("handleIndex: error in marshaling json: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.tmpl.Execute(w, struct {
		Stations  []*Station
		Fields    []string
		Landuse   []string
		Map       string
		StartDate string
		EndDate   string
	}{
		stations,
		f.Fields,
		f.Landuse,
		string(mapJSON),
		time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
	})
	if err != nil {
		log.Printf("handleIndex: error in executing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	opts := &Filter{}
	err := json.NewDecoder(r.Body).Decode(opts)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Printf("handleUpdate: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	acl := []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"}
	if err := opts.Validate(acl); err != nil {
		log.Printf("handleUpdate: %v\n", err)
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	f, err := s.db.Filter(opts)
	if err != nil {
		log.Printf("handleUpdate: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(f)
	if err != nil {
		log.Printf("handleUpdate: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		log.Printf("handleSeries: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	opts, err := NewSeriesOptionsFromForm(r)
	if err != nil {
		log.Printf("handleSeries: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b, err := s.db.Series(opts)
	if err != nil {
		log.Printf("handleSeries: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.csv", time.Now().Unix())
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+f)

	csv.NewWriter(w).WriteAll(b)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set default security headers.
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "deny")

	s.mux.ServeHTTP(w, r)
}
