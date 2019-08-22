// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// Station defines metadata about a physical station.
type Station struct {
	ID        int64
	Name      string
	Landuse   string
	Altitude  int64
	Latitude  float64
	Longitude float64
}

func (s *Station) UnmarshalJSON(b []byte) error {
	var l snipeit.Location
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}

	s.ID = l.ID
	s.Name = l.Name
	s.Landuse = l.Currency
	s.Altitude, _ = strconv.ParseInt(l.Zip, 10, 64)
	s.Latitude, _ = strconv.ParseFloat(l.Address, 64)
	s.Longitude, _ = strconv.ParseFloat(l.Address2, 64)

	return nil
}

type Response struct {
	Fields   []string
	Stations []string
	Landuse  []string

	snipeitRef []int64
}

type QueryOptions struct {
	Fields   []string
	Stations []string
	Landuse  []string
	From     string
	To       string
}

func (q *QueryOptions) Query() (string, error) {
	tmpl := `SHOW TAG VALUES FROM {{ if .Fields }} {{  join .Fields "," }} {{ else }} /.*/ {{ end }} WITH KEY IN ("station", "landuse", "snipeit_location_ref"){{ if .Where }} WHERE {{ join .Where " OR " }} {{ end }}`

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
