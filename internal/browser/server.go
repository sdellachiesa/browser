// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
	"time"

	"gitlab.inf.unibz.it/lter/browser/static"
)

type Server struct {
	db  Backend
	mux *http.ServeMux
}

func NewServer(b Backend) *Server {
	s := &Server{
		db:  b,
		mux: http.NewServeMux(),
	}

	s.mux.HandleFunc("/", s.handleIndex())
	s.mux.Handle("/static/", static.Handler())
	// TODO
	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})

	s.mux.HandleFunc("/api/v1/update", s.handleUpdate)
	s.mux.HandleFunc("/api/v1/series/", s.handleSeries)

	return s
}

func (s *Server) handleIndex() http.HandlerFunc {
	tmplFile, err := static.File("static/index.html")
	if err != nil {
		log.Fatalf("handleIndex: error in reading template: %v", err)
	}

	funcMap := template.FuncMap{
		"Landuse":     MapLanduse,
		"Measurement": MapMeasurements,
	}

	tmpl, err := template.New("base").Funcs(funcMap).Parse(tmplFile)
	if err != nil {
		log.Fatalf("handleIndex: error in parsing template: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
		opts := &Filter{
			Fields: []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"},
		}
		f, err := s.db.Get(opts)
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

		err = tmpl.Execute(w, struct {
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
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
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

	f, err := s.db.Get(opts)
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
	s.mux.ServeHTTP(w, r)
}
