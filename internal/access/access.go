// Copyright 2019 Eurac Research. All rights reserved.

// Package access parses a JSON access control file.
//
// An access file is composed of several access rules. An access rule has a
// unique name and an access control list for controlling access to specific
// fields of data for measurements, stations and landuse.
//
// An example of an access file is presented below:
// 	[
//		{
//			"name": "Public",
//			"acl": {
//				"measurements": ["a"],
//				"stations": [1, 2],
//				"landuse": ["me"]
//		},
//		{
//			"name": "FullAccess",
//			"acl": {
//				"measurements": [],
//				"stations": [],
//				"landuse": []
//		}
//	]
//
package access

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
)

const DefautlRefreshInterval = 10 * time.Minute

var (
	// Guarantee we implement browser.Database.
	_ browser.Database = &Access{}

	// Guarantee we implement browser.Metadata.
	_ browser.Metadata = &Access{}

	// ErrNoRuleFound means that no rule was found for the
	// given name.
	ErrNoRuleFound = errors.New("access: no rule found")

	// identifier is a regular expression used for checking
	// if a given user input is a valid influx identifier.
	identifier = regexp.MustCompile(`^\w+$`)

	// defaultRule is the rule which will return on any kind
	// of error, so we always ensure a rule is returned.
	defaultRule = &Rule{
		Name: browser.Public,
		ACL: &AccessControlList{
			Measurements: []string{
				"air_t_avg",
				"air_rh_avg",
				"wind_dir",
				"wind_speed_avg",
				"wind_speed_max",
				"wind_speed",
				"nr_up_sw_avg",
				"precip_rt_nrt_tot",
				"snow_height"},
		},
	}
)

// Access represents a parsed JSON Access file.
type Access struct {
	db       browser.Database
	metadata browser.Metadata

	mu    sync.RWMutex // guards the fields below
	last  time.Time
	rules []*Rule
}

// Rule represents a single access rule.
type Rule struct {
	Name browser.Role
	ACL  *AccessControlList
}

// AccessControlList represents an access list.
type AccessControlList struct {
	Measurements []string
	Stations     []string
	Landuse      []string
}

// New returns a new instance of Access.
func New(file string, db browser.Database, m browser.Metadata) (*Access, error) {
	a := &Access{
		db:       db,
		metadata: m,
	}

	// append build in default rules
	a.rules = append(a.rules,
		defaultRule,
		&Rule{
			Name: browser.FullAccess,
			ACL:  &AccessControlList{},
		},
	)

	if err := a.loadRules(file); err != nil {
		return nil, err
	}

	go a.refreshRules(file)

	return a, nil

}

func (a *Access) Series(ctx context.Context, m *browser.Message) (browser.TimeSeries, error) {
	a.redact(ctx, m)
	return a.db.Series(ctx, m)
}

func (a *Access) Query(ctx context.Context, m *browser.Message) *browser.Stmt {
	a.redact(ctx, m)
	return a.db.Query(ctx, m)
}

func (a *Access) Stations(ctx context.Context, m *browser.Message) (browser.Stations, error) {
	a.redact(ctx, m)
	return a.metadata.Stations(ctx, m)
}

func (a *Access) redact(ctx context.Context, m *browser.Message) {
	u := browser.UserFromContext(ctx)
	rule := a.rule(u.Role)

	m.Landuse = a.clear(m.Landuse, rule.ACL.Landuse)
	m.Measurements = a.clear(m.Measurements, rule.ACL.Measurements)
	m.Stations = a.clear(m.Stations, rule.ACL.Stations)
}

// clear clears not allowed fields and returns a new slice.
func (a *Access) clear(input, allowed []string) []string {
	if len(input) == 0 {
		return allowed
	}

	m := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		m[v] = struct{}{}
	}

	var c []string
	for _, v := range input {
		if ok := identifier.MatchString(v); !ok {
			continue
		}

		_, ok := m[v]
		if !ok && len(m) > 0 {
			continue
		}

		c = append(c, v)
	}

	if len(c) == 0 {
		return allowed
	}

	return c
}

// rule returns a rule form the given role. If no rule is
// found or it's ACL is nil a default hardcoded rule will be
// returned.
func (a *Access) rule(role browser.Role) *Rule {
	if role == "" {
		return defaultRule
	}

	a.mu.RLock()
	rules := a.rules
	a.mu.RUnlock()

	for _, r := range rules {
		if r.Name == role && r.ACL != nil {
			return r
		}
	}

	return defaultRule
}

// loadRules loads rules from the given file.
func (a *Access) loadRules(file string) error {
	fi, err := os.Stat(file)
	if os.IsNotExist(err) {
		log.Printf("access: no access file %q found, use build in rules.\n", file)
		return nil
	}
	if err != nil {
		return fmt.Errorf("access: %v", err)
	}

	mtime := fi.ModTime()
	if !mtime.After(a.last) && a.rules != nil {
		return nil // no changes to rules file
	}

	f, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("access: error in opening %q: %v", file, err)
	}

	var r []*Rule
	if err := json.Unmarshal(f, &r); err != nil {
		return fmt.Errorf("access: error in JSON decoding rules file %q: %v", file, err)
	}

	a.mu.Lock()
	a.last = mtime
	a.rules = r
	a.mu.Unlock()

	log.Printf("access: update access rules from file %q\n", file)

	return nil
}

// refreshRules refreshes the rules from the given file by
// the DefautlRefreshInterval.
func (a *Access) refreshRules(file string) {
	ticker := time.NewTicker(DefautlRefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := a.loadRules(file); err != nil {
			log.Println(err)
		}
	}
}
