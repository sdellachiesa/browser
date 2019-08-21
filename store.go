// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/influx"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// The Backend interface retrieves data and return a []byte.
type Backend interface {
	Stations(*QueryOptions) (map[string]*Station, error)
	Fields(*QueryOptions) ([]string, error)
	Series(*QueryOptions) ([][]string, error)
}

type Datastore struct {
	snipeit *snipeit.Client
	influx  *influx.Client
}

func NewDatastore(sc *snipeit.Client, ic *influx.Client) Backend {
	return Datastore{sc, ic}
}

func (d Datastore) Fields(opts *QueryOptions) ([]string, error) {
	q := "show measurements"

	if len(opts.Fields) > 0 {
		q = fmt.Sprintf("%s with measurement =~ /%s/", q, strings.Join(opts.Fields, "|"))
	}

	if len(opts.Stations) >= 1 {
		w := []string{}
		for _, s := range opts.Stations {
			w = append(w, fmt.Sprintf("station='%s'", s))
		}
		q = fmt.Sprintf("%s WHERE %s", q, strings.Join(w, " AND "))
	}

	log.Println(q)

	result, err := d.influx.Result(q)
	if err != nil {
		return nil, err
	}

	fields := []string{}
	for _, r := range result.Series {
		for _, v := range r.Values {
			fields = append(fields, v[0].(string))
		}
	}

	return fields, nil
}

func (d Datastore) Series(opts *QueryOptions) ([][]string, error) {
	// TODO: QueryOptions should implement a "Queryer" interface which
	// provides a method Query.
	s := []string{}
	for _, f := range opts.Stations {
		s = append(s, fmt.Sprintf("station='%s'", f))
	}
	q := fmt.Sprintf("SELECT station,landuse,altitude,latitude,longitude,%s FROM %s WHERE %s AND time >= '%s' AND time <= '%s' GROUP BY station",
		strings.Join(opts.Fields, ","),
		strings.Join(opts.Fields, ","),
		strings.Join(s, " OR "),
		opts.From,
		opts.To,
	)

	log.Println(q)

	results, err := d.influx.Results(q)
	if err != nil {
		return nil, err
	}

	type key struct {
		station   string
		timestamp string
	}

	header := []string{}
	values := make(map[key][]string)
	for _, result := range results {
		for i, serie := range result.Series {
			if i == 0 {
				header = serie.Columns
			}

			for _, value := range serie.Values {
				k := key{serie.Tags["station"], fmt.Sprint(value[0])}
				column, ok := values[k]
				if !ok {
					column = make([]string, len(value))
				}

				for i := range value {
					v := value[i]
					if v == nil {
						continue
					}

					if i == 0 {
						var err error
						v, err = time.Parse(time.RFC3339, value[0].(string))
						if err != nil {
							log.Printf("cannot convert timestamp: %v. skipping.", err)
							continue
						}
					}

					column[i] = fmt.Sprint(v)
				}

				values[k] = column
			}
		}
	}

	rows := [][]string{}
	rows = append(rows, header)
	for _, v := range values {
		rows = append(rows, v)
	}

	return rows, nil
}

func (d Datastore) Stations(opts *QueryOptions) (map[string]*Station, error) {
	q, err := opts.Query()
	if err != nil {
		return nil, err
	}
	log.Println(q)
	result, err := d.influx.Result(q)
	if err != nil {
		return nil, err
	}

	stations := make(map[string]*Station)
	for _, serie := range result.Series {
		s, ok := stations[serie.Tags["station"]]
		if !ok {
			s = &Station{
				Name: serie.Tags["station"],
			}
		}
		s.Measurements = append(s.Measurements, serie.Name)

		for _, v := range serie.Values {
			for i := range v {
				switch i {
				case 0: // skip timestamp
					continue
				case 1:
					n, _ := v[i].(json.Number).Int64()
					s.Altitude = n
				case 2:
					n, _ := v[i].(json.Number).Float64()
					s.Latitude = n
				case 3:
					n, _ := v[i].(json.Number).Float64()
					s.Longitude = n
				case 4:
					s.Landuse = v[i].(string)
				}
			}
		}

		stations[serie.Tags["station"]] = s
	}

	return stations, nil
}
