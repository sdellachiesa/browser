// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package influx provides the implementation of the browser.Database interface
// using InfluxDB as backend. It uses an in-memory cache based on maps guarded
// by a sync.RWMutex for storing data of queries like "SHOW TAG VALUES" which
// can be slow on large datasets.
package influx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/ql"

	client "github.com/influxdata/influxdb1-client/v2"
)

var (
	// Guarantee we implement browser.Series.
	_ browser.Database = &DB{}

	// CacheRefreshInterval is the interval in which the cache will be refreshed.
	CacheRefreshInterval = 8 * time.Hour

	// groupRegexpMap maps a Group to a regular expression for matching
	// measurements.
	groupRegexpMap = map[browser.Group]*regexp.Regexp{
		browser.AirTemperature:                               regexp.MustCompile(`^air_t(.*)*$`),
		browser.RelativeHumidity:                             regexp.MustCompile(`^air_rh(.*)*$`),
		browser.SoilTemperature:                              regexp.MustCompile(`^st_.*|_st_.*$`),
		browser.SoilTemperatureDepth00:                       regexp.MustCompile(`^(.*st_.*|_st_).*_00_.*$`),
		browser.SoilTemperatureDepth02:                       regexp.MustCompile(`^(.*st_.*|_st_).*_02_.*$`),
		browser.SoilTemperatureDepth05:                       regexp.MustCompile(`^(.*st_.*|_st_).*_05_.*$`),
		browser.SoilTemperatureDepth10:                       regexp.MustCompile(`^(.*st_.*|_st_).*_10_.*$`),
		browser.SoilTemperatureDepth20:                       regexp.MustCompile(`^(.*st_.*|_st_).*_20_.*$`),
		browser.SoilTemperatureDepth40:                       regexp.MustCompile(`^(.*st_.*|_st_).*_40_.*$`),
		browser.SoilTemperatureDepth50:                       regexp.MustCompile(`^(.*st_.*|_st_).*_50_.*$`),
		browser.SoilWaterContent:                             regexp.MustCompile(`^swc_[^dp_|ec_|st_]`),
		browser.SoilWaterContentDepth00:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_00_.*$`),
		browser.SoilWaterContentDepth02:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_02_.*$`),
		browser.SoilWaterContentDepth05:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_05_.*$`),
		browser.SoilWaterContentDepth10:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_10_.*$`),
		browser.SoilWaterContentDepth20:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_20_.*$`),
		browser.SoilWaterContentDepth40:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_40_.*$`),
		browser.SoilWaterContentDepth50:                      regexp.MustCompile(`^swc_[^dp_|ec_|st_].*_50_.*$`),
		browser.SoilElectricalConductivity:                   regexp.MustCompile(`^swc_ec_.*$`),
		browser.SoilElectricalConductivityDepth00:            regexp.MustCompile(`^swc_ec_.*00_.*$`),
		browser.SoilElectricalConductivityDepth02:            regexp.MustCompile(`^swc_ec_.*02_.*$`),
		browser.SoilElectricalConductivityDepth05:            regexp.MustCompile(`^swc_ec_.*05_.*$`),
		browser.SoilElectricalConductivityDepth10:            regexp.MustCompile(`^swc_ec_.*10_.*$`),
		browser.SoilElectricalConductivityDepth20:            regexp.MustCompile(`^swc_ec_.*20_.*$`),
		browser.SoilElectricalConductivityDepth40:            regexp.MustCompile(`^swc_ec_.*40_.*$`),
		browser.SoilElectricalConductivityDepth50:            regexp.MustCompile(`^swc_ec_.*50_.*$`),
		browser.SoilDielectricPermittivity:                   regexp.MustCompile(`^swc_dp_.*$`),
		browser.SoilDielectricPermittivityDepth00:            regexp.MustCompile(`^swc_dp_.*00_.*$`),
		browser.SoilDielectricPermittivityDepth02:            regexp.MustCompile(`^swc_dp_.*02_.*$`),
		browser.SoilDielectricPermittivityDepth05:            regexp.MustCompile(`^swc_dp_.*05_.*$`),
		browser.SoilDielectricPermittivityDepth10:            regexp.MustCompile(`^swc_dp_.*10_.*$`),
		browser.SoilDielectricPermittivityDepth20:            regexp.MustCompile(`^swc_dp_.*20_.*$`),
		browser.SoilDielectricPermittivityDepth40:            regexp.MustCompile(`^swc_dp_.*40_.*$`),
		browser.SoilDielectricPermittivityDepth50:            regexp.MustCompile(`^swc_dp_.*50_.*$`),
		browser.SoilWaterPotential:                           regexp.MustCompile(`^swp.[^_st_].*$`),
		browser.SoilWaterPotentialDepth00:                    regexp.MustCompile(`^swp.[^_st_].*_00_.*$`),
		browser.SoilWaterPotentialDepth02:                    regexp.MustCompile(`^swp.[^_st_].*_02_.*$`),
		browser.SoilWaterPotentialDepth05:                    regexp.MustCompile(`^swp.[^_st_].*_05_.*$`),
		browser.SoilWaterPotentialDepth10:                    regexp.MustCompile(`^swp.[^_st_].*_10_.*$`),
		browser.SoilWaterPotentialDepth20:                    regexp.MustCompile(`^swp.[^_st_].*_20_.*$`),
		browser.SoilWaterPotentialDepth40:                    regexp.MustCompile(`^swp.[^_st_].*_40_.*$`),
		browser.SoilWaterPotentialDepth50:                    regexp.MustCompile(`^swp.[^_st_].*_50_.*$`),
		browser.SoilHeatFlux:                                 regexp.MustCompile(`^shf.*$`),
		browser.SoilSurfaceTemperature:                       regexp.MustCompile(`.*surf_t.*$`), // TODO: "surf_t_" and not("mv")
		browser.WindSpeed:                                    regexp.MustCompile(`^wind_speed.*$`),
		browser.WindSpeedAvg:                                 regexp.MustCompile(`^wind_speed.*_avg$`),
		browser.WindSpeedMax:                                 regexp.MustCompile(`^wind_speed.*_max$`),
		browser.WindDirection:                                regexp.MustCompile(`^wind_dir`),
		browser.Precipitation:                                regexp.MustCompile(`^precip.*(_tot|_int).*$`),
		browser.PrecipitationTotal:                           regexp.MustCompile(`^precip.*(_tot).*$`),
		browser.PrecipitationIntensity:                       regexp.MustCompile(`^precip.*(_int).*$`),
		browser.SnowHeight:                                   regexp.MustCompile(`snow_height`),
		browser.LeafWetnessDuration:                          regexp.MustCompile(`^lwm`),
		browser.SunshineDuration:                             regexp.MustCompile(`^sun`),
		browser.PhotosyntheticallyActiveRadiation:            regexp.MustCompile(`^par_.*$`),
		browser.PhotosyntheticallyActiveRadiationTotal:       regexp.MustCompile(`^par_.*(dif_|soil_).*$`),
		browser.PhotosyntheticallyActiveRadiationDiffuse:     regexp.MustCompile(`^par_.*dif_.*$`),
		browser.PhotosyntheticallyActiveRadiationAtSoilLevel: regexp.MustCompile(`^par_.*soil_.*$`),
		browser.NDVIRadiations:                               regexp.MustCompile(`^ndvi_.*`),
		browser.PRIRadiations:                                regexp.MustCompile(`^pri_.*$`),
		browser.ShortWaveRadiation:                           regexp.MustCompile(`^sr_|.*_sw_.*$`),
		browser.ShortWaveRadiationIncoming:                   regexp.MustCompile(`^.*_dn.*_sw_.*$`),
		browser.ShortWaveRadiationOutgoing:                   regexp.MustCompile(`^.*_up.*_sw_.*$`),
		browser.LongWaveRadiation:                            regexp.MustCompile(`.*_lw_.*$`),
		browser.LongWaveRadiationIncoming:                    regexp.MustCompile(`.*_dn.*_lw_.*$`),
		browser.LongWaveRadiationOutgoing:                    regexp.MustCompile(`.*_up.*_lw_.*$`),
	}
)

// DB holds information for communicating with InfluxDB.
type DB struct {
	client   client.Client
	database string

	mu                     sync.RWMutex // guards the fields below
	stationGroupsCache     map[int64][]browser.Group
	groupMeasurementsCache map[browser.Group][]string // will contain only measurements which are not maintenance
}

// NewDB returns a new instance of DB and initializes the internal caches and
// starts a new go routine for refreshing the cache on the defined
// CacheRefreshInterval.
func NewDB(client client.Client, database string) (*DB, error) {
	db := &DB{
		client:             client,
		database:           database,
		stationGroupsCache: make(map[int64][]browser.Group),
	}

	if err := db.loadCache(); err != nil {
		return nil, err
	}
	go db.refreshCache()

	return db, nil
}

// loadCache initializes a in memory cache due to the slowness of metadata
// queries like "SHOW TAG VALUES" on large datasets inside InfluxDB.
func (db *DB) loadCache() error {
	resp, err := db.exec(ql.ShowTagValues().From().WithKeyIn("snipeit_location_ref"))
	if err != nil {
		return err
	}

	gCache := make(map[int64][]browser.Group)
	mCache := make(map[browser.Group][]string)
	for _, result := range resp.Results {
		for _, series := range result.Series {
			// add series name to list of measurements if it doesn't belong to
			// maintenance.
			if isAllowed(series.Name, maintenace) {
				continue
			}

			// Match series.Name parent groups
			g := matchGroupByType(series.Name, browser.ParentGroup)

			// Match series.Name to sub groups too.
			sg := matchGroupByType(series.Name, browser.SubGroup)

			for _, value := range series.Values {
				id, err := strconv.ParseInt(value[1].(string), 10, 64)
				if err == nil {
					gCache[id] = browser.AppendGroupIfMissing(gCache[id], g)
					gCache[id] = browser.AppendGroupIfMissing(gCache[id], sg)
				}
			}

			mCache[g] = browser.AppendStringIfMissing(mCache[g], series.Name)
			mCache[sg] = browser.AppendStringIfMissing(mCache[sg], series.Name)
		}
	}

	db.mu.Lock()
	db.stationGroupsCache = gCache
	db.groupMeasurementsCache = mCache
	db.mu.Unlock()

	log.Println("influx: caches initialized")
	return nil
}

// matchGroupByType returns a group for the given label. A return of NoGroup indicates
// no match.
func matchGroupByType(label string, t browser.GroupType) browser.Group {
	for _, group := range browser.GroupsByType(t) {
		re, ok := groupRegexpMap[group]
		if !ok {
			continue
		}

		if re.MatchString(label) {
			return group
		}
	}

	return browser.NoGroup
}

func (db *DB) refreshCache() {
	ticker := time.NewTicker(CacheRefreshInterval)

	for range ticker.C {
		if err := db.loadCache(); err != nil {
			log.Println(err)
		}
		log.Println("influx: caches updated")
	}
}

func (db *DB) GroupsByStation(ctx context.Context, id int64) ([]browser.Group, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user := browser.UserFromContext(ctx)
	groups, ok := db.stationGroupsCache[id]
	if ok {
		return browser.FilterGroupsByRole(groups, user.Role), nil
	}

	return []browser.Group{}, browser.ErrGroupsNotFound
}

func (db *DB) Maintenance(ctx context.Context) ([]string, error) {
	user := browser.UserFromContext(ctx)
	if user.Role != browser.FullAccess && !user.License {
		return []string{}, nil
	}
	return maintenace, nil
}

func (db *DB) Series(ctx context.Context, filter *browser.SeriesFilter) (browser.TimeSeries, error) {
	if filter == nil {
		return nil, browser.ErrDataNotFound
	}

	resp, err := db.exec(db.seriesQuery(ctx, filter))
	if err != nil {
		return nil, err
	}

	var ts browser.TimeSeries
	for _, result := range resp.Results {
		for _, series := range result.Series {
			nTime := filter.Start

			m := &browser.Measurement{
				Label:       series.Name,
				Aggregation: series.Tags["aggr"],
				Unit:        series.Tags["unit"],
				Station: &browser.Station{
					Name:    series.Tags["station"],
					Landuse: series.Tags["landuse"],
				},
			}

			for _, value := range series.Values {
				t, err := time.ParseInLocation(time.RFC3339, value[0].(string), time.UTC)
				if err != nil {
					log.Printf("cannot convert timestamp: %v. skipping.", err)
					continue
				}

				// Fill missing timestamps with NaN values, to return a time
				// series with a continuous time range. The interval of raw data
				// in LTER is 15 minutes. See:
				// https://gitlab.inf.unibz.it/lter/browser/issues/10
				for !t.Equal(nTime) {
					m.Points = append(m.Points, &browser.Point{
						Timestamp: nTime,
						Value:     math.NaN(),
					})
					nTime = nTime.Add(browser.DefaultCollectionInterval)
				}
				nTime = t.Add(browser.DefaultCollectionInterval)

				f, err := value[1].(json.Number).Float64()
				if err != nil {
					log.Printf("cannot convert value to float: %v. skipping.", err)
					continue
				}

				// Add additional metadata only on the first run.
				m.Station.Elevation, err = value[2].(json.Number).Int64()
				if err != nil {
					m.Station.Elevation = -1
				}

				m.Station.Latitude, err = value[3].(json.Number).Float64()
				if err != nil {
					m.Station.Latitude = -1.0
				}

				m.Station.Longitude, err = value[4].(json.Number).Float64()
				if err != nil {
					m.Station.Longitude = -1.0
				}

				if value[5] == nil {
					m.Depth = 0
				} else {
					m.Depth, err = value[5].(json.Number).Int64()
					if err != nil {
						m.Depth = -1
					}
				}
				p := &browser.Point{
					Timestamp: t,
					Value:     f,
				}
				m.Points = append(m.Points, p)
			}

			ts = append(ts, m)
		}
	}

	return ts, nil
}

func (db *DB) seriesQuery(ctx context.Context, filter *browser.SeriesFilter) ql.Querier {
	return ql.QueryFunc(func() (string, []interface{}) {
		var (
			buf          bytes.Buffer
			args         []interface{}
			start, end   = startEndTime(filter.Start, filter.End)
			user         = browser.UserFromContext(ctx)
			measurements = db.parseMeasurements(ctx, filter)
		)

		// If the users has full access and the filter contains maintenance
		// measurements add them to the slice.
		if user.Role == browser.FullAccess && user.License {
			measurements = appendMaintenance(measurements, filter.Maintenance...)
		}

		for _, measure := range measurements {
			columns := []string{measure, "altitude as elevation", "latitude", "longitude", "depth"}

			sb := ql.Select(columns...)
			sb.From(measure)
			sb.Where(
				ql.Eq(ql.Or(), "snipeit_location_ref", filter.Stations...),
				ql.And(),
				ql.TimeRange(start, end),
			)
			sb.GroupBy("station,snipeit_location_ref,landuse,unit,aggr")
			sb.OrderBy("time").ASC().TZ("Etc/GMT-1")

			q, arg := sb.Query()
			buf.WriteString(q)
			buf.WriteString(";")

			args = append(args, arg)
		}

		return buf.String(), args
	})
}

// appendMaintenance appends the given labels to s if the label is present in
// the maintenance slice.
func appendMaintenance(s []string, label ...string) []string {
	for _, l := range label {
		for _, m := range maintenace {
			if strings.EqualFold(l, m) {
				s = append(s, l)
			}
		}
	}

	return s
}

// Data in InfluxDB is UTC but LTER data is UTC+1 therefor we need to adapt
// start and end times. It will shift the start time to -1 hour and will set
// the end time to 22:59:59 in order to capture a full day.
func startEndTime(s time.Time, e time.Time) (time.Time, time.Time) {
	start := s.Add(-1 * time.Hour)
	end := time.Date(e.Year(), e.Month(), e.Day(), 22, 59, 59, 59, time.UTC)
	return start, end
}

func (db *DB) Query(ctx context.Context, filter *browser.SeriesFilter) *browser.Stmt {
	var measures []string
	if len(filter.Groups) > 0 {
		measures = db.parseMeasurements(ctx, filter)
	}

	measures = appendMaintenance(measures, filter.Maintenance...)

	c := []string{"station", "landuse", "altitude as elevation", "latitude", "longitude"}
	c = append(c, measures...)

	start, end := startEndTime(filter.Start, filter.End)

	q, _ := ql.Select(c...).From(measures...).Where(
		ql.Eq(ql.Or(), "snipeit_location_ref", filter.Stations...),
		ql.And(),
		ql.TimeRange(start, end),
	).OrderBy("time").ASC().TZ("Etc/GMT-1").Query()

	return &browser.Stmt{
		Query:    q,
		Database: db.database,
	}
}

// parseMeasurements will return a list of InfluxDB measurements, read from
// cache, by the given filter. It will remove measurements based on the user
// role.
func (db *DB) parseMeasurements(ctx context.Context, filter *browser.SeriesFilter) []string {
	db.mu.RLock()
	cache := db.groupMeasurementsCache
	db.mu.RUnlock()

	var (
		labels []string
		user   = browser.UserFromContext(ctx)
	)
	for _, group := range filter.Groups {
		measurements, ok := cache[group]
		if !ok {
			continue
		}

		for _, m := range measurements {
			// check if the user is allowed to retrieve the measurement. If not
			// continue. This is the minimum on access control which is present.
			// Only registered and signed users have access to the full data
			// set.
			if user.Role == browser.Public && !isAllowed(m, publicAllowed) {
				continue
			}

			// Only include std if explicitly declared in the filter.
			if strings.HasSuffix(m, "_std") && !filter.WithSTD {
				continue
			}

			labels = browser.AppendStringIfMissing(labels, m)
		}
	}

	sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })

	return labels
}

// exec executes the given ql query and returns a response.
func (db *DB) exec(q ql.Querier) (*client.Response, error) {
	query, _ := q.Query()

	if query == "" {
		return nil, errors.New("db.exec: given query is empty")
	}

	resp, err := db.client.Query(client.NewQuery(query, db.database, ""))
	if err != nil {
		return nil, fmt.Errorf("db.exec: %v", err)
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("db.exec: %v", resp.Error())
	}

	return resp, nil
}

func isAllowed(label string, allowed []string) bool {
	for _, f := range allowed {
		if strings.EqualFold(label, f) {
			return true
		}
	}
	return false
}
