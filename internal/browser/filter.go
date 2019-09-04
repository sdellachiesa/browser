// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
)

type Filter struct {
	Fields   []string
	Stations []string
	Landuse  []string
}

// TODO: Thats ugly but for know and for the RC for IBK it does the job.
func (f *Filter) Validate(d []string) error {
	if len(f.Fields) == 0 {
		f.Fields = d
		return nil
	}

	for _, field := range f.Fields {
		if !IsAllowed(field) {
			return fmt.Errorf("error: field name %q not allowed", field)
		}

		if !In(field, d) {
			return errors.New("field not found")
		}
	}

	for _, station := range f.Stations {
		if !IsAllowed(station) {
			return errors.New("station not allowed")
		}
	}

	for _, landuse := range f.Landuse {
		if !IsAllowed(landuse) {
			return errors.New("landuse not allowed")
		}
	}

	return nil
}

func (f *Filter) Query() (string, error) {
	tmpl := `SHOW TAG VALUES FROM{{ if .Fields }} {{  join .Fields "," }} {{ else }} /.*/ {{ end }}WITH KEY IN ("landuse", "snipeit_location_ref"){{ if .Where }} WHERE {{ join .Where " OR " }}{{ end }}`

	funcMap := template.FuncMap{
		"join": strings.Join,
	}

	where := []string{}
	for _, s := range f.Stations {
		where = append(where, fmt.Sprintf("snipeit_location_ref='%s'", s))
	}
	for _, l := range f.Landuse {
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
		f.Fields,
		where,
	}); err != nil {
		return "", fmt.Errorf("could not apply InfluxQL query data: %v ", err)
	}

	return b.String(), nil
}

func IsAllowed(s string) bool {
	ok, err := regexp.MatchString(`^\w+$`, s)
	if err != nil {
		log.Println(err)
		return false
	}
	return ok
}

func In(v string, s []string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}

	return false
}
