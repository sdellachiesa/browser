package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser/static"

	client "github.com/influxdata/influxdb1-client/v2"
)

type server struct {
	db       client.Client
	database string
	mux      *http.ServeMux
	// key to prevent request forgery; static for server's lifetime.
	//key string
}

func newServer(options ...func(s *server) error) (*server, error) {
	s := &server{mux: http.NewServeMux()}

	for _, o := range options {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	if s.db == nil {
		return nil, fmt.Errorf("must provide an option func that specifies a store")
	}

	s.mux.HandleFunc("/", s.handleStatic)
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/_api", s.handleAPI)

	return s, nil
}

func (s *server) handleStatic(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path[1:]
	if p == "" {
		p = "index.html"
	}

	b, err := static.File(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, path.Base(p), time.Now(), strings.NewReader(b))
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ServeHTTP satisfies the http.Handler interface.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	method := r.FormValue("method")
	log.Println(method)
	var (
		resp interface{}
		err  error
	)
	switch method {
	case "stations": // return station names
		response, err := s.db.Query(client.NewQuery("SHOW TAG VALUES FROM /.*/ WITH KEY = station", s.database, ""))
		if err != nil || response.Error() != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, result := range response.Results {
			resp = result.Series
		}
	case "fields":
		q := "SHOW MEASUREMENTS"

		stations := r.FormValue("stations")
		if stations != "" {
			q = fmt.Sprintf("SHOW MEASUREMENTS WHERE station='%s'", stations)
		}

		response, err := s.db.Query(client.NewQuery(q, s.database, ""))
		if err != nil || response.Error() != nil {
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		for _, result := range response.Results {
			resp = result.Series
		}
	case "query":
		query := `
SELECT {{ if .select }} {{- .select -}} {{ else }} * {{ end }}
FROM {{ if .from }} {{-  .from -}} {{ else }} /.*/ {{ end }}
{{ if .where -}}
WHERE {{.where}}
{{ end }}`
		data := map[string]interface{}{
			"from":   r.FormValue("stations"),
			"select": r.FormValue("fields"),
			"where":  strings.Join([]string{"landuse = $1 OR landuse = $2", "time >= $3 AND time <= $4", "altitue >= $5 AND altitude <= $6"}, " AND "),
		}

		q, err := prepareQuery(query, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(q)

		resp = struct {
			Query string
		}{q}
	default:
		resp = struct {
			Error string
		}{"no method passed"}
	}

	j, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(j)

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