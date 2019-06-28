// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Station defines metadata about a physical station.
type Station struct {
	Name         string
	Landuse      string
	Altitude     int64
	Latitude     float64
	Longitude    float64
	Measurements []string
}

type QueryOptions struct {
	Fields   []string
	Stations []string
	Landuse  []string
	From     string
	To       string
}

func (q *QueryOptions) Query() (string, error) {
	tmpl := `SELECT altitude,latitude,longitude,landuse 
FROM {{ if .Fields }} {{  join .Fields "," }} {{ else }} record {{ end }}
{{ if .Where }} WHERE {{ join .Where " OR " }} {{ end }}
GROUP BY station LIMIT 1`

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	where := []string{}
	for _, s := range q.Stations {
		where = append(where, fmt.Sprintf("station='%s'", s))
	}
	for _, l := range q.Landuse {
		where = append(where, fmt.Sprintf("landuse='%s'", l))
	}

	t, err := template.New("query").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("could not parse InfluxQL query template: %v ", err)
	}

	var b bytes.Buffer
	if err := t.Execute(&b, struct {
		Fields []string
		Where  []string
	}{
		q.Fields,
		where,
	}); err != nil {
		return "", fmt.Errorf("could not apply InfluxQL query data: %v ", err)
	}

	return b.String(), nil
}
