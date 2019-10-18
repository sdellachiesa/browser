// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"fmt"
	"log"
	"net/http"
	"sort"
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

// key is used as map key for sorting and grouping the map entries.
type key struct {
	station string

	// timestamp is a UNIX epoch and not of a time.Time since
	// the later should be avoid for map keys.
	// Read https://golang.org/src/time/time.go?#L101
	timestamp int64
}

// Time returns the key's unix timestamp as time.Time.
func (k key) Time() time.Time {
	timeLoc := time.FixedZone("UTC+1", 60*60)
	return time.Unix(k.timestamp, 0).In(timeLoc)
}

// Next returns a new key with it's timestamp modified by the given duration.
func (k key) Next(d time.Duration) key {
	return key{k.station, k.Time().Add(d).Unix()}
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

	var (
		table  = make(map[key][]string)
		keys = []key{}
		header = []string{}
	)
	for _, result := range resp.Results {
		for i, serie := range result.Series {
			if i == 0 {
				header = serie.Columns
			}

			for _, value := range serie.Values {
				ts, err := time.ParseInLocation(time.RFC3339, value[0].(string), time.UTC)
				if err != nil {
					log.Printf("cannot convert timestamp: %v. skipping.", err)
					continue
				}

				k := key{serie.Tags["snipeit_location_ref"], ts.Unix()}

				column, ok := table[k]
				if !ok {
					keys = append(keys, k)

					// Initialize column and fill it with NaN's.
					column = make([]string, len(value))
					for i := range column {
						column[i] = "NaN"
					}
				}

				for i := range value {
					v := value[i]
					if v == nil {
						continue
					}

					// The value at index 0 corresponds to the timestamp.
					if i == 0 {
						v = ts.Format("2006-01-02 15:04:05")
					}

					column[i] = fmt.Sprint(v)

				}

				table[k] = column
			}
		}
	}

	// Sort by timesstamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].timestamp < keys[j].timestamp
	})

	// Sort by station,timestamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].station < keys[j].station
	})

	var (
		rows = [][]string{header}
		last = keys[len(keys)-1]
	)
	for _, k := range keys {
		c, ok := table[k]
		if !ok {
			continue
		}
		rows = append(rows, c)

		// Fill up missing timestamps in order to have a continuous timerange
		// otherwise Researchers cannot work with the data and will complain.
		// What a bummer. The interval of 'RAW' data in LTER is always 15 Minutes.
		// See: https://gitlab.inf.unibz.it/lter/browser/issues/10
		next := k.Next(15 * time.Minute)
		for {
			// Check if there is a record for the next timestamp if yes
			// break out of the for loop. Otherwise stay in the loop and
			// fill up missing rows until the next real entry.
			_, ok := table[next]
			if ok || next.Time().After(last.Time()) {
				break
			}

			// Initialize column and fill it with NaN's.
			column := make([]string, len(c))
			for i := range column {
				column[i] = "NaN"
			}
			column[0] = next.Time().Format("2006-01-02 15:04:05")
			column[1] = c[1]

			rows = append(rows, column)
			next = next.Next(15 * time.Minute)
		}
	}

	return rows, nil
}

// Stations retrieves all stations which make part of the LTER project from SnipeIT and
// returns them. It will filter for the given ID's if provided.
func (d Datastore) Stations(ids ...string) ([]*Station, error) {
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
