// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

	s.mux.Handle("/", static.Handler())

	s.mux.HandleFunc("/api/v1/fields/", s.handleFields)
	s.mux.HandleFunc("/api/v1/stations/", s.handleStations)
	s.mux.HandleFunc("/api/v1/series/", s.handleSeries)

	return s
}

func (s *Server) handleStations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	opts := &QueryOptions{}
	err := json.NewDecoder(r.Body).Decode(opts)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Printf("handleStations: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	if len(opts.Fields) == 0 {
		opts.Fields = []string{"t_air", "air_t", "tair", "rh", "air_rh", "wind_dir", "mean_wind_direction", "wind_speed_avg", "mean_wind_speed", "wind_speed_max"}
	}

	resp, err := s.db.Stations(opts)
	if err != nil {
		log.Printf("handleStations: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("handleStations: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func (s *Server) handleFields(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	opts := &QueryOptions{}
	err := json.NewDecoder(r.Body).Decode(opts)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Printf("handleFields: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	if len(opts.Fields) == 0 {
		log.Println("yess")
		opts.Fields = []string{"^t_air$", "^air_t$", "^tair$", "^rh$", "^air_rh$", "^wind_dir$", "^mean_wind_direction$", "^wind_speed_avg$", "^mean_wind_speed$", "^wind_speed_max$"}
	}

	resp, err := s.db.Fields(opts)
	if err != nil {
		log.Printf("handleFields: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		log.Printf("handleFields: %v\n", err)
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

	opts := &QueryOptions{
		Fields:   r.Form["fields"],
		Stations: r.Form["stations"],
		Landuse:  r.Form["landuse"],
		From:     r.FormValue("startDate"),
		To:       r.FormValue("endDate"),
	}

	b, err := s.db.Series(opts)
	if err != nil {
		log.Printf("handleSeries: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f := fmt.Sprintf("LTER_%d.csv", time.Now().Unix())
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+f)

	csv.NewWriter(w).WriteAll(b)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
