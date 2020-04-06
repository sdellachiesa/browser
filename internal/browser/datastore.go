// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"

	"github.com/euracresearch/go-snipeit"
	client "github.com/influxdata/influxdb1-client/v2"
)

var ErrDataNotFound = errors.New("no data points")

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

// Datastore implements the Backend interface and talks to InfluxDB
// and SnipeIT and handles all data authorization.
type Datastore struct {
	snipeit *snipeit.Client

	influx   client.Client
	database string

	access Authorizer

	mu    sync.RWMutex
	cache map[string]Stations
}

// NewDatastore returns a new datastore, initializes it's cache and
// keeps updating the cache every 12*time.Hour.
func NewDatastore(s *snipeit.Client, i client.Client, database string, acl Authorizer) (*Datastore, error) {
	d := &Datastore{
		snipeit:  s,
		influx:   i,
		database: database,
		access:   acl,
		cache:    make(map[string]Stations),
	}

	if err := d.loadCache(); err != nil {
		return nil, err
	}

	go d.refreshCache(12 * time.Hour)

	return d, nil
}

// loadCache initializes the datastore's cache for each ACL rule due
// to dhe slow "SHOW TAG VALUES" queries on large datasets inside
// InfluxDB.
func (d *Datastore) loadCache() error {
	cache := make(map[string]Stations)

	for _, n := range d.access.Names() {
		rule := d.access.Rule(n)

		var where ql.Querier
		if len(rule.ACL.Stations) > 0 {
			where = ql.Eq(ql.Or(), "snipeit_location_ref", rule.ACL.Stations...)
		}
		if len(rule.ACL.Landuse) > 0 {
			where = ql.Eq(ql.Or(), "landuse", rule.ACL.Landuse...)
		}

		q := ql.ShowTagValues().From(rule.ACL.Measurements...).WithKeyIn("snipeit_location_ref").Where(where)
		resp, err := d.exec(q)
		if err != nil {
			return err
		}

		stations, err := d.stations(rule.ACL.Stations...)
		if err != nil {
			return err
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

		cache[rule.Name] = stations.WithMeasurements()
	}

	d.mu.Lock()
	d.cache = cache
	d.mu.Unlock()

	return nil
}

// refreshCache refreshes the server cache on the given interval.
func (d *Datastore) refreshCache(i time.Duration) {
	for {
		if err := d.loadCache(); err != nil {
			log.Println(err)
		}
		time.Sleep(i)
	}
}

// exec executes the given ql querie and returns a response.
func (d *Datastore) exec(q ql.Querier) (*client.Response, error) {
	query, _ := q.Query()

	resp, err := d.influx.Query(client.NewQuery(query, d.database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	return resp, nil
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

	c := []string{"station", "landuse", "altitude as elevation", "latitude", "longitude"}
	c = append(c, req.measurements...)

	q, _ := ql.Select(c...).From(req.measurements...).Where(
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
			columns := []string{"station", "landuse", "altitude as elevation", "latitude", "longitude"}
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

func (d *Datastore) units(req *request, h []string) ([]string, error) {
	units := make([]string, len(h))

	q := ql.ShowTagValues().From(req.measurements...).WithKeyIn("unit").Where(ql.TimeRange(req.start, req.end))
	resp, err := d.exec(q)
	if err != nil {
		return units, err
	}

	m := make(map[string]string)
	for _, result := range resp.Results {
		for _, serie := range result.Series {
			for _, value := range serie.Values {
				m[serie.Name] = value[1].(string)
			}
		}
	}

	for i, v := range h {
		u, ok := m[v]
		if !ok {
			continue
		}

		units[i] = u
	}

	return units, nil
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

	resp, err := d.exec(d.seriesQuery(req))
	if err != nil {
		return nil, err
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

	if len(keys) == 0 {
		return nil, ErrDataNotFound
	}

	// Sort by timesstamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].timestamp < keys[j].timestamp
	})

	// Sort by station,timestamp.
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].station < keys[j].station
	})

	units, _ := d.units(req, header)
	rows := [][]string{header, units}
	last := keys[len(keys)-1]

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
	if len(a) == 0 {
		return true
	}

	for _, v := range a {
		if v == s {
			return true
		}
	}

	return false
}
