// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/influx"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
)

// The Backend interface retrieves data.
type Backend interface {
	Get(*Filter) (*Response, error)
	Series(*SeriesOptions) ([][]string, error)
	StationsMetadata(ids []int64) ([]*Station, error)
}

type Datastore struct {
	snipeit *snipeit.Client
	influx  *influx.Client
}

func NewDatastore(sc *snipeit.Client, ic *influx.Client) Backend {
	return Datastore{sc, ic}
}

func (d Datastore) Get(opts *Filter) (*Response, error) {
	q, err := opts.Query()
	if err != nil {
		return nil, err
	}

	log.Println(q)

	result, err := d.influx.Result(q)
	if err != nil {
		return nil, err
	}

	resp := &Response{}
	for _, s := range result.Series {
		resp.Fields = append(resp.Fields, s.Name)

		for _, v := range s.Values {
			key, value := v[0].(string), v[1].(string)
			switch key {
			case "snipeit_location_ref":
				id, _ := strconv.ParseInt(value, 10, 64)
				resp.Stations = append(resp.Stations, id)
			case "landuse":
				resp.Landuse = append(resp.Landuse, value)
			}
		}
	}

	resp.Landuse = unique(resp.Landuse)

	return resp, nil
}

func unique(s []string) []string {
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}

	return s[:j]
}

func (d Datastore) Series(opts *SeriesOptions) ([][]string, error) {
	q, err := opts.Query()
	if err != nil {
		return nil, err
	}

	results, err := d.influx.Results(q)
	if err != nil {
		return nil, err
	}

	type key struct {
		station   string
		timestamp string
	}

	values := make(map[key][]string)
	keys := []key{}
	header := []string{}
	for _, result := range results {
		for i, serie := range result.Series {
			if i == 0 {
				header = serie.Columns
			}

			for _, value := range serie.Values {
				k := key{serie.Tags["station"], value[0].(string)}

				column, ok := values[k]
				if !ok {
					keys = append(keys, k)
					column = make([]string, len(value))
				}

				for i := range value {
					v := value[i]
					if v == nil {
						continue
					}

					// The value at index 0 corresponds to the timestamp
					if i == 0 {
						ts, err := time.Parse(time.RFC3339, v.(string))
						if err != nil {
							log.Printf("cannot convert timestamp: %v. skipping.", err)
							continue
						}
						// Timestamps in InfluxDB are in UTC, but station records are in UTC+1
						// so we need to add +1h offset to the parsed time.
						v = ts.Add(1 * time.Hour).Format("2006-01-02 15:04:05")
					}

					column[i] = fmt.Sprint(v)
				}

				values[k] = column
			}
		}
	}

	rows := [][]string{}
	rows = append(rows, header)
	for i := 0; i < len(keys); i++ {
		v := values[keys[i]]
		rows = append(rows, v)
	}

	return rows, nil
}

// StationsMetadata returns all metadata associated with a station stored
// in SnipeIT. It will filter for stations with the given ids.
func (d Datastore) StationsMetadata(ids []int64) ([]*Station, error) {
	u, err := d.snipeit.AddOptions("locations", &snipeit.LocationOptions{Search: "LTER"})
	if err != nil {
		return nil, err
	}

	req, err := d.snipeit.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Total int64
		Rows  []*Station
	}
	_, err = d.snipeit.Do(req, &response)
	if err != nil {
		return nil, err
	}

	stations := []*Station{}
	for _, s := range response.Rows {
		if inArray(s.ID, ids) {
			stations = append(stations, s)
		}
	}

	return stations, nil
}

func inArray(i int64, a []int64) bool {
	if a == nil {
		return true
	}

	for _, v := range a {
		if v == i {
			return true
		}
	}

	return false
}
