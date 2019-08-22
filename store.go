// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/influx"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// The Backend interface retrieves data and return a []byte.
type Backend interface {
	Get(*QueryOptions) (*Response, error)
	Series(*QueryOptions) ([][]string, error)
}

type Datastore struct {
	snipeit *snipeit.Client
	influx  *influx.Client
}

func NewDatastore(sc *snipeit.Client, ic *influx.Client) Backend {
	return Datastore{sc, ic}
}

type Response struct {
	Fields   []string
	Stations []string
	Landuse  []string
}

func (d Datastore) Get(opts *QueryOptions) (*Response, error) {
	q, err := opts.Query()
	if err != nil {
		return nil, err
	}

	result, err := d.influx.Result(q)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]struct{})
	stations := make(map[string]struct{})
	landuse := make(map[string]struct{})
	for _, s := range result.Series {
		fields[s.Name] = struct{}{}
		for _, v := range s.Values {
			key, value := v[0].(string), v[1].(string)
			switch key {
			case "station":
				stations[value] = struct{}{}
			case "landuse":
				landuse[value] = struct{}{}
			}
		}
	}

	resp := &Response{}

	resp.Stations, err = Keys(stations)
	if err != nil {
		return nil, err
	}
	resp.Landuse, err = Keys(landuse)
	if err != nil {
		return nil, err
	}
	resp.Fields, err = Keys(fields)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func Keys(v interface{}) ([]string, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, errors.New("error not a map")
	}
	t := rv.Type()
	if t.Key().Kind() != reflect.String {
		return nil, errors.New("not string key")
	}
	var result []string
	for _, kv := range rv.MapKeys() {
		result = append(result, kv.String())
	}
	sort.Strings(result)
	return result, nil
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
