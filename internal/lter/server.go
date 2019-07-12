package lter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	client "github.com/influxdata/influxdb1-client/v2"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
	"gitlab.inf.unibz.it/lter/browser/static"
)

type Backend struct {
	Influx  client.Client
	SnipeIT *snipeit.Client

	Database string
}

type Server struct {
	db  *Backend
	mux *http.ServeMux
}

func NewServer(b *Backend) *Server {
	s := &Server{
		db:  b,
		mux: http.NewServeMux(),
	}

	s.mux.Handle("/", static.Handler())

	s.mux.HandleFunc("/api/v1/stations/", s.handleStations)
	s.mux.HandleFunc("/api/v1/measurements/", s.handleMeasurements)
	s.mux.HandleFunc("/api/v1/series/", s.handleSeries)
	s.mux.HandleFunc("/api/v1/metadata/", s.handleMetadata)
	s.mux.HandleFunc("/api/v1/templates/", s.handleTemplates)

	return s
}

func (s *Server) key(r *http.Request) string {
	_, p := s.mux.Handler(r)
	return r.URL.Path[len(p):]
}

func (s *Server) handleStations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		l   []*snipeit.Location
		err error
	)
	key := s.key(r)
	if key != "" {
	} else {
		l, _, err = s.db.SnipeIT.Locations(&snipeit.LocationOptions{key})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}

	j, err := json.MarshalIndent(l, " ", "	")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (s *Server) handleMeasurements(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (s *Server) handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

// ServeHTTP satisfies the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func prepareQuery(tmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("query").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("could not parse SQL query template: %v ", err)
	}

	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("could not apply SQL query data: %v ", err)
	}

	return b.String(), nil
}