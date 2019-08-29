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
		log.Fatal(err)
	}

	funcMap := template.FuncMap{
		"Landuse": MapLanduse,
	}

	tmpl, err := template.New("base").Funcs(funcMap).Parse(tmplFile)
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Hardcode for presentation in IBK will be replaced by the ACL middleware.
		opts := &FilterOptions{
			Fields: []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"},
		}
		response, err := s.db.Get(opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stations, err := s.db.StationsMetadata(response.Stations)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		mapJSON, err := json.Marshal(stations)
		if err != nil {
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
			response.Fields,
			response.Landuse,
			string(mapJSON),
			time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
			time.Now().Format("2006-01-02"),
		})
		if err != nil {
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

	opts := &FilterOptions{}
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
		opts.Fields = []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"}
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

// TODO: This should be replace when we introduce i18n
func MapLanduse(key string) string {
	switch key {
	case "pa":
		return "Pasture"
	case "me":
		return "Meadow"
	case "fo":
		return "Forest"
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
