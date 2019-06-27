package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	client "github.com/influxdata/influxdb1-client/v2"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

type Backend struct {
	Influx  client.Client
	SnipeIT *snipeit.Client

	Database string
}

type APIHandler struct {
	db  *Backend
	mux *http.ServeMux
}

func NewAPIHandler(b *Backend) *APIHandler {
	a := &APIHandler{
		db:  b,
		mux: http.NewServeMux(),
	}

	a.mux.HandleFunc("/api/v1/stations/", a.handleStations)
	a.mux.HandleFunc("/api/v1/measurements/", a.handleMeasurements)
	a.mux.HandleFunc("/api/v1/series/", a.handleSeries)
	a.mux.HandleFunc("/api/v1/metadata/", a.handleMetadata)
	a.mux.HandleFunc("/api/v1/templates/", a.handleTemplates)

	return a
}

func (a *APIHandler) key(r *http.Request) string {
	_, p := a.mux.Handler(r)
	return r.URL.Path[len(p):]
}

func (a *APIHandler) handleStations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var (
		l   []*snipeit.Location
		err error
	)
	key := a.key(r)
	if key != "" {
	} else {
		l, _, err = a.db.SnipeIT.Locations(&snipeit.LocationOptions{key})
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

func (a *APIHandler) handleMeasurements(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (a *APIHandler) handleSeries(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (a *APIHandler) handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

func (a *APIHandler) handleTemplates(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("not yet implemented"))
}

// ServeHTTP satisfies the http.Handler interface.
func (a *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
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
