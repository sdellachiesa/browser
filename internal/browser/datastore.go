// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/ql"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"

	client "github.com/influxdata/influxdb1-client/v2"
)

// Datastore denotes a backend which is composed of SnipeIT
// and InfluxDB.
type Datastore struct {
	snipeit  *snipeit.Client
	influx   client.Client
	database string
}

// NewDatastore returns a new datastore.
func NewDatastore(sc *snipeit.Client, ic client.Client, database string) *Datastore {
	return &Datastore{
		snipeit:  sc,
		influx:   ic,
		database: database,
	}
}

func (d Datastore) Filter(q ql.Querier) (*Filter, error) {
	query, _ := q.Query()

	log.Println(query)

	resp, err := d.influx.Query(client.NewQuery(query, d.database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	f := &Filter{}
	for _, result := range resp.Results {
		for _, s := range result.Series {
			f.Fields = append(f.Fields, s.Name)

			for _, v := range s.Values {
				key, value := v[0].(string), v[1].(string)
				switch key {
				case "snipeit_location_ref":
					f.Stations = appendIfMissing(f.Stations, value)
				case "landuse":
					f.Landuse = appendIfMissing(f.Landuse, value)
				}
			}
		}
	}
	return f, nil
}

func appendIfMissing(slice []string, s string) []string {
	for _, el := range slice {
		if el == s {
			return slice
		}
	}
	return append(slice, s)
}

func (d Datastore) Series(q ql.Querier) ([][]string, error) {
	query, _ := q.Query()

	log.Println(query)

	resp, err := d.influx.Query(client.NewQuery(query, d.database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	type key struct {
		station   string
		timestamp string
	}

	values := make(map[key][]string)
	keys := []key{}
	header := []string{}
	for _, result := range resp.Results {
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
func (d Datastore) Stations(ids []string) ([]*Station, error) {
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

func inArray(s string, a []string) bool {
	if a == nil {
		return true
	}

	for _, v := range a {
		if v == s {
			return true
		}
	}

	return false
}
