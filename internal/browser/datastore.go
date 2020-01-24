// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"

	client "github.com/influxdata/influxdb1-client/v2"
)

// Authorizer is an interface for handling authorization to the LTER data.
type Authorizer interface {
	// Filter filters out not permitted or not authorized values for the given
	// context.
	Filter(ctx context.Context, r *request) error

	// Names lists all names of registered authorization rules.
	Names() []string

	// Rule returns the rule by the given name.
	Rule(name string) *Rule
}

// Datastore denotes a backend which is composed of SnipeIT
// and InfluxDB.
type Datastore struct {
	snipeit *snipeit.Client

	influx   client.Client
	database string

	access Authorizer

	mu    sync.RWMutex
	cache map[string]Stations
}

// NewDatastore returns a new datastore and populates its cache.
func NewDatastore(s *snipeit.Client, i client.Client, database string, acl Authorizer) (*Datastore, error) {
	d := &Datastore{
		snipeit:  s,
		influx:   i,
		database: database,
		access:   acl,
		cache:    make(map[string]Stations),
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

// init initializes the cache for each ACL rule due to dhe slow "SHOW
// TAG VALUES" queries on large datasets.
func (d *Datastore) init() error {
	for _, n := range d.access.Names() {
		rule := d.access.Rule(n)

		stations, err := d.stations(rule.ACL.Stations...)
		if err != nil {
			return err
		}

		var where ql.Querier
		if len(rule.ACL.Stations) > 0 {
			where = ql.Eq(ql.Or(), "snipeit_location_ref", rule.ACL.Stations...)
		}
		if len(rule.ACL.Landuse) > 0 {
			where = ql.Eq(ql.Or(), "landuse", rule.ACL.Landuse...)
		}

		q := ql.ShowTagValues().From(rule.ACL.Measurements...).WithKeyIn("snipeit_location_ref").Where(where)
		query, _ := q.Query()

		resp, err := d.influx.Query(client.NewQuery(query, d.database, ""))
		if err != nil {
			return err
		}
		if resp.Error() != nil {
			return fmt.Errorf("%v", resp.Error())
		}

		for _, result := range resp.Results {
			for _, s := range result.Series {
				for _, v := range s.Values {
					id := v[1].(string)

					station, ok := stations.Get(id)
					if !ok {
						continue
					}

					station.Measurements = append(station.Measurements, s.Name)
				}
			}
		}

		d.cache[rule.Name] = stations
	}

	return nil
}

func (d *Datastore) Get(role auth.Role) Stations {
	d.mu.RLock()
	defer d.mu.RUnlock()

	s, ok := d.cache[string(role)]
	if !ok {
		return d.cache[defaultRule.Name]
	}

	return s
}

func (d *Datastore) Query(ctx context.Context, req *request) string {
	d.access.Filter(ctx, req)

	columns := []string{"station", "landuse", "altitude", "latitude", "longitude"}
	columns = append(columns, req.measurements...)

	q, _ := ql.Select(columns...).From(req.measurements...).Where(
		ql.Eq(ql.Or(), "snipeit_location_ref", req.stations...),
		ql.And(),
		ql.TimeRange(req.start, req.end),
	).OrderBy("time").ASC().TZ("Etc/GMT-1").Query()

	return q
}

func (d *Datastore) seriesQuery(req *request) ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		var (
			buf  bytes.Buffer
			args []interface{}
		)
		for _, station := range req.stations {
			columns := []string{"station", "landuse", "altitude", "latitude", "longitude"}
			columns = append(columns, req.measurements...)

			sb := ql.Select(columns...)
			sb.From(req.measurements...)
			sb.Where(
				ql.Eq(ql.And(), "snipeit_location_ref", station),
				ql.And(),
				ql.TimeRange(req.start, req.end),
			)
			sb.GroupBy("station,snipeit_location_ref")
			sb.OrderBy("time").ASC().TZ("Etc/GMT-1")

			q, arg := sb.Query()
			buf.WriteString(q)
			buf.WriteString(";")

			args = append(args, arg)
		}

		return buf.String(), args
	})
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

func (d *Datastore) Series(ctx context.Context, req *request) ([][]string, error) {
	d.access.Filter(ctx, req)

	query, _ := d.seriesQuery(req).Query()
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
		keys   = []key{}
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

// stations retrieves all stations which make part of the LTER project
// from SnipeIT and returns them. It will filter for the given ID's
// if provided.
func (d *Datastore) stations(ids ...string) (Stations, error) {
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

	var stations Stations
	for _, s := range response.Rows {
		if s.Name == "LTER" {
			continue
		}

		if inArray(s.ID, ids) {
			stations = append(stations, s)
		}
	}

	// Sort stations by name.
	sort.Slice(stations, func(i, j int) bool {
		return stations[i].Name < stations[j].Name
	})

	return stations, nil
}

func inArray(s string, a []string) bool {
	if a == nil || len(a) == 0 {
		return true
	}

	for _, v := range a {
		if v == s {
			return true
		}
	}

	return false
}
