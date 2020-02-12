// Copyright 2019 Eurac Research. All rights reserved.
package browser

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

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
)

// TODO: Currently start/end times are not handled by the ACL system.

var defaultRule = &Rule{
	Name: "Public",
	ACL: &AccessControlList{
		Measurements: []string{
			"air_t_avg",
			"air_rh_avg",
			"wind_dir",
			"wind_speed_avg",
			"wind_speed_max",
			"wind_speed",
			"nr_up_sw_avg",
			"sr_avg",
			"precip_rt_nrt_tot",
			"snow_height"},
	},
}

// ErrNoRuleFound means that no rule was found for the given name.
var ErrNoRuleFound = errors.New("access: no rule found")

// identifier is a regular expression used for checking if a given
// user input is a valid influx identifier.
var identifier = regexp.MustCompile(`^\w+$`)

// Access represents a parsed JSON Access file, which is composed of
// several rules. An access rule has a unique name and an access list
// for controling the access to sepcific fields of the data. These
// fields can be measurements, stations and landuse. If a field is
// empty or missing full access to to it will be granted.
//
// An example of an access file is presented below:
// 	[
//		{
//			"name": "FullAccess",
//			"acl": {
//				"measurements": ["a"],
//				"stations": ["c"],
//				"landuse": [],
//		}
//	]
type Access struct {
	mu    sync.RWMutex // guards the fields below
	last  time.Time
	rules []*Rule
}

// Rule represents a single access rule.
type Rule struct {
	Name string
	ACL  *AccessControlList
}

// AccessControlList represents an access list.
type AccessControlList struct {
	Measurements []string
	Stations     []string
	Landuse      []string
}

// ParseAccessFile parses the content of the given file and returns
// the parsed Access.  On an interval of 10*time.Minutes it will check
// for changes made to the given file and update the parsed Access
// if necessary.
func ParseAccessFile(file string) *Access {
	a := &Access{}
	if err := a.loadRules(file); err != nil {
		log.Fatal(err)
	}
	go a.refreshRules(file)

	return a
}

// enforce will check the given input values if they are valid
// identifiers and permitted by the given ACL values. Not permitted
// values will be filtered out and a new slice containing the valid
// values will be returned.
func (a *Access) enforce(input, allowed []string) []string {
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

func (a *Access) Filter(ctx context.Context, r *request) error {
	role, ok := ctx.Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	rule := a.Rule(string(role))

	r.measurements = a.enforce(r.measurements, rule.ACL.Measurements)
	r.stations = a.enforce(r.stations, rule.ACL.Stations)
	r.landuse = a.enforce(r.landuse, rule.ACL.Landuse)

	return nil
}

// Rule returns a rule form the given name. If no rule is found or
// it's ACL is nil a default hardcoded rule will be returned.
func (a *Access) Rule(name string) *Rule {
	if name == "" {
		return defaultRule
	}

	a.mu.RLock()
	rules := a.rules
	a.mu.RUnlock()

	for _, r := range rules {
		if r.Name == name && r.ACL != nil {
			return r
		}
	}

	return defaultRule
}

// Names returns a slice of names of all complete rules.
func (a *Access) Names() []string {
	a.mu.RLock()
	rules := a.rules
	a.mu.RUnlock()

	var n []string
	for _, r := range rules {
		if r.ACL == nil || r.Name == "" {
			continue
		}
		n = append(n, r.Name)
	}

	return n
}

// loadRules loads rules from the given file.
func (a *Access) loadRules(file string) error {
	fi, err := os.Stat(file)
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
	return nil
}

// refreshRules refreshes the rules from the given file every
// 10 minutes.
func (a *Access) refreshRules(file string) {
	for {
		if err := a.loadRules(file); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Minute * 10)
	}
}
