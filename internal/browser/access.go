// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
)

// ErrNoRuleFound means that no rule was found for the given name.
var ErrNoRuleFound = errors.New("access: no  rule found")

// Access represents a parsed JSON Access file. The file should have the following
// format:
// [
//      {
//		"roleName": "FullAccess",
//		"policy": {
//			"fields": ["a"],
//			"stations": ["c"],
//			"landuse": []
//	},
//	{
//		...
//	}
// ]
//
type Access struct {
	mu    sync.RWMutex // guards the fields below
	last  time.Time
	rules []*Rule
}

// Rule represents a single rule which applies to a specific role.
type Rule struct {
	RoleName auth.Role
	Policy   *Filter
}

// ParseAccessFile parses the content of the given file and returns the parsed Access.
// On an interval of 10*time.Minutes it will check for changes made to the given
// file and update the parsed Access if necessary.
func ParseAccessFile(file string) *Access {
	a := &Access{}
	if err := a.loadRules(file); err != nil {
		log.Fatal(err)
	}
	go a.refreshRules(file)

	return a
}

// DecodeAndValidate implements the Decoder interface.
func (a *Access) DecodeAndValidate(r *http.Request) (*Filter, error) {
	rule, err := a.Rule(r.Context())
	if err != nil {
		return nil, err
	}

	f := &Filter{}
	switch r.Header.Get("content-type") {
	case "application/x-www-form-urlencoded; charset=UTF-8":
		fallthrough
	case "application/x-www-form-urlencoded": // FORM Submit
		f, err = a.decodeForm(r)
		if err != nil {
			return nil, err
		}
	default: // JSON
		err := json.NewDecoder(r.Body).Decode(&f)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
	}

	f.Fields = a.inputFilter(f.Fields, rule.Policy.Fields)
	f.Stations = a.inputFilter(f.Stations, rule.Policy.Stations)
	f.Landuse = a.inputFilter(f.Landuse, rule.Policy.Landuse)

	return f, nil
}

// inputFilter will check the given input values if they are valid
// identifiers and permitted by the given allowed values. Not permitted
// values will be filtered out and a new slice containing the valid values
// will be returned.
func (a *Access) inputFilter(input, allowed []string) []string {
	if len(input) == 0 {
		return allowed
	}

	m := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		m[strings.ToLower(v)] = struct{}{}
	}

	var c []string
	for _, v := range input {
		ok, err := regexp.MatchString(`^\w+$`, v)
		if err != nil || !ok {
			continue
		}

		_, ok = m[v]
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

// deocde data from a form post.
func (a *Access) decodeForm(r *http.Request) (*Filter, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	var err error
	f := &Filter{}

	f.start, err = time.Parse("2006-01-02", r.FormValue("startDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}
	// In order to start the day at 00:00:00
	f.start = f.start.Add(-1 * time.Hour)

	f.end, err = time.Parse("2006-01-02", r.FormValue("endDate"))
	if err != nil {
		return nil, fmt.Errorf("error: could not parse end date %v", err)
	}

	if f.end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	// Limit download of data to one year
	limit := time.Date(f.end.Year()-1, f.end.Month(), f.end.Day(), 0, 0, 0, 0, time.UTC)
	if f.start.Before(limit) {
		return nil, errors.New("error: time range is greater then a year")
	}

	f.Fields = r.Form["fields"]
	if f.Fields == nil {
		return nil, errors.New("error: at least one field must be given")
	}

	f.Stations = r.Form["stations"]
	if f.Stations == nil {
		return nil, errors.New("error: at least one station must be given")
	}

	f.Landuse = r.Form["landuse"]

	return f, nil
}

// Rule returns a rule from the given context. If no rule is found it will try to find and return the
// default rule. If that failes it will return a basic hardcoded rule.
func (a *Access) Rule(ctx context.Context) (*Rule, error) {
	role, ok := ctx.Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	r, err := a.find(role)
	if err != nil {
		return &Rule{
			RoleName: auth.Public,
			Policy: &Filter{
				Fields: []string{"air_t_avg", "air_rh_avg", "wind_dir", "wind_speed_avg", "wind_speed_max"},
			},
		}, nil
	}

	return r, nil
}

func (a *Access) find(name auth.Role) (*Rule, error) {
	a.mu.RLock()
	rules := a.rules
	a.mu.RUnlock()

	for _, r := range rules {
		if r.RoleName == name {
			return r, nil
		}
	}

	return nil, ErrNoRuleFound
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

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("access: error in opening %q: %v", file, err)
	}
	defer f.Close()

	var r []*Rule
	if err := json.NewDecoder(f).Decode(&r); err != nil {
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
