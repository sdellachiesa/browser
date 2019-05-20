package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
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

// TODO: Should we only return a http.Handler?
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
	log.Println("hit api handler Say yes")
	method := r.FormValue("method")
	log.Println(method)
	var (
		resp interface{}
		err  error
	)
	switch method {
	case "stations":
		landuse := r.FormValue("landuse")
		fields := r.FormValue("fields")

		if landuse != "" {
			landuse = fmt.Sprintf(" WHERE landuse =~ /%s/", strings.ReplaceAll(landuse, ",", "|"))
		}

		if fields != "" {
			q := fmt.Sprintf("SELECT %s FROM /.*/ LIMIT 1", fields)
			response, err := s.db.Query(client.NewQuery(q, s.database, ""))
			if err != nil || response.Error() != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var f []string
			for _, result := range response.Results {
				for _, row := range result.Series {
					f = append(f, row.Name)
				}
			}
			fields = fmt.Sprintf("FROM %s", strings.Join(f, ","))
		}

		q := fmt.Sprintf("SHOW TAG VALUES %s WITH KEY =~ /.*/%s", fields, landuse)
		log.Println(q)
		response, err := s.db.Query(client.NewQuery(q, s.database, ""))
		if err != nil || response.Error() != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, result := range response.Results {
			resp = result.Series
		}
	case "fields":
		stations := r.FormValue("stations")
		if stations != "" {
			stations = fmt.Sprintf("FROM %s", stations)
		}

		q := fmt.Sprintf("SHOW FIELD KEYS %s", stations)
		log.Println(q)
		response, err := s.db.Query(client.NewQuery(q, s.database, ""))
		if err != nil || response.Error() != nil {
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}

		for _, result := range response.Results {
			resp = result.Series
		}
	case "series":
		where := []string{}
		stations := r.FormValue("stations")
		if stations == "" {
			stations = "/.*/"
		}
		fields := r.FormValue("fields")
		if fields == "" {
			fields = "*"
		}
		sDate := r.FormValue("start")
		if sDate != "" {
			where = append(where, fmt.Sprintf("time >= '%s'", sDate))
		}
		eDate := r.FormValue("end")
		if eDate != "" {
			where = append(where, fmt.Sprintf("time <= '%s'", eDate))
		}
		landuse := r.FormValue("landuse")
		if landuse != "" {
			where = append(where, fmt.Sprintf("landuse =~ /%s/", landuse))
		}
		altitude := strings.Split(r.FormValue("altitude"), ";")
		if len(altitude) == 2 {
			aMin, err := strconv.ParseInt(altitude[0], 10, 64)
			if err != nil {
				break
			}
			aMax, err := strconv.ParseInt(altitude[1], 10, 64)
			if err != nil {
				break
			}
			where = append(where, fmt.Sprintf("altitude >= %d AND altitude <= %d", aMin, aMax))
		}

		filter := ""
		if len(where) > 0 {
			filter = "WHERE " + strings.Join(where, " AND ")
		}

		q := fmt.Sprintf("SELECT %s FROM %s %s", fields, stations, filter)
		log.Println(q)
		response, err := s.db.Query(client.NewQuery(q, s.database, ""))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if response.Error() != nil {
			http.Error(w, response.Error().Error(), http.StatusInternalServerError)
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