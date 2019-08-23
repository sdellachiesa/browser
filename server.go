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

	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.Handle("/static/", static.Handler())
	// TODO
	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})

	s.mux.HandleFunc("/api/v1/update", s.handleUpdate)
	s.mux.HandleFunc("/api/v1/series/", s.handleSeries)

	return s
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	opts := &QueryOptions{
		Fields: []string{"t_air", "air_t", "tair", "rh", "air_rh", "wind_dir", "mean_wind_direction", "wind_speed_avg", "mean_wind_speed", "wind_speed_max"},
	}
	resp, err := s.db.Get(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d, err := s.db.Stations(resp.snipeitRef)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapdata, err := json.Marshal(d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := static.File("static/base.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"Landuse": MapLanduse,
	}

	t, err := template.New("base").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, struct {
		MapData string
		*Response
	}{fmt.Sprintf("%s", mapdata), resp})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("handleUpdate: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
	if len(opts.Fields) == 0 {
		opts.Fields = []string{"t_air", "air_t", "tair", "rh", "air_rh", "wind_dir", "mean_wind_direction", "wind_speed_avg", "mean_wind_speed", "wind_speed_max"}
	}

	d, err := s.db.Get(opts)
	if err != nil {
		log.Println(err)
	}

	b, err := json.Marshal(d)
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

// TODO: This should be replace when we introduce i18n
func MapLanduse(key string) string {
	switch key {
	case "pa":
		return "Pasture"
	case "me":
		return "Meadow"
	case "fo":
		return "Climate station in the forest"
	case "sf":
		return "SapFlow"
	case "de":
		return "Dendrometer"
	case "ro":
		return "Rock"
	case "bs":
		return "Bare soil"
	default:
		return key
	}
}
