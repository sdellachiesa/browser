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
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

// Decoder is an interface for decoding data.
type Decoder interface {
	// DecodeAndValidate decodes data from the given HTTP request and
	// validates it and returns an ql.Querier.
	DecodeAndValidate(r *http.Request) (ql.Querier, error)
}

// RequestDecoder is a decoder which validates and restricts access
// to data depending on the role a request/user makes part of.
type RequestDecoder struct {
	mu    sync.RWMutex // guards the fields below
	last  time.Time
	rules []*Rule
}

// Rule denotes a simple rule which applies to a specific role.
type Rule struct {
	Role   string
	Policy *Filter
}

// NewRequestDecoder returns a new RequestDecoder which will
// parse rules from the given file. On a fixed interval of
// 10 minutes it will check if the rule file has changed and
// if so it will update the rules.
// The file should be a JSON file with the following layout:
//
// [
//      {
//		"role": "FullAccess",
//		"policy": {
//			"fields": [],
//			"stations": [],
//			"landuse": []
//		}
// ]
//
func NewRequestDecoder(file string) *RequestDecoder {
	rd := &RequestDecoder{}
	if err := rd.loadRules(file); err != nil {
		log.Fatal(err)
	}
	go rd.refreshRules(file)

	return rd
}

// DecodeAndValidate takes the given HTTP request decodes and validates it.
func (rd *RequestDecoder) DecodeAndValidate(r *http.Request) (ql.Querier, error) {
	rule, err := rd.Rule(r.Context())
	if err != nil {
		return nil, err
	}

	f := &Filter{}
	switch r.Header.Get("content-type") {
	case "application/x-www-form-urlencoded": // FORM Submit
		f, err = rd.decodeForm(r)
		if err != nil {
			return nil, err
		}

		f.qType = SeriesQuery
	default: // JSON
		err := json.NewDecoder(r.Body).Decode(&f)
		if err == io.EOF {
			err = nil
		}
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		f.qType = UpdateQuery
	}

	f.Fields = rd.inputFilter(f.Fields, rule.Policy.Fields)
	f.Stations = rd.inputFilter(f.Stations, rule.Policy.Stations)
	f.Landuse = rd.inputFilter(f.Landuse, rule.Policy.Landuse)

	return f, nil
}

// inputFilter will check the given input values if they are valid
// identifiers and permitted by the given allowed values. Not permitted
// values will be filtered out and a new slice containing the valid values
// will be returned.
func (rd *RequestDecoder) inputFilter(input, allowed []string) []string {
	if len(input) == 0 {
		return allowed
	}

	m := make(map[string]struct{}, len(allowed))
	for _, v := range allowed {
		m[v] = struct{}{}
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
func (rd *RequestDecoder) decodeForm(r *http.Request) (*Filter, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	start, err := time.Parse("2006-01-02", r.FormValue("startDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.Parse("2006-01-02", r.FormValue("endDate"))
	if err != nil {
		return nil, fmt.Errorf("error: could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	// Limit download of data to one year
	limit := time.Date(end.Year()-1, end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if start.Before(limit) {
		return nil, errors.New("error: time range is greater then a year")
	}

	if r.Form["fields"] == nil {
		return nil, errors.New("error: at least one field must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("error: at least one station must be given")
	}

	return &Filter{
		Fields:   r.Form["fields"],
		Stations: r.Form["stations"],
		Landuse:  r.Form["landuse"],

		// We need to shift the timerange of one hour since in influx we use UTC and in output we want
		// UTC+1.
		start: time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC).Add(-1 * time.Hour),
		end:   time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.UTC).Add(-1 * time.Hour),
	}, nil
}

// Rule returns a rule from the given context. If no rule is found
// it will try to find and return the default rule.
func (rd *RequestDecoder) Rule(ctx context.Context) (*Rule, error) {
	role, ok := ctx.Value(auth.JWTClaimsContextKey).(string)
	if !ok {
		return rd.findDefault()
	}

	return rd.find(role)
}

func (rd *RequestDecoder) findDefault() (*Rule, error) {
	return rd.find("Public")
}

func (rd *RequestDecoder) find(name string) (*Rule, error) {
	rd.mu.RLock()
	rules := rd.rules
	rd.mu.RUnlock()

	for _, r := range rules {
		if r.Role == name {
			return r, nil
		}
	}

	return nil, fmt.Errorf("No rule with name %q policy found.", name)
}

// loadRules loads rules from the given file.
func (rd *RequestDecoder) loadRules(file string) error {
	fi, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("validator: %v", err)
	}
	mtime := fi.ModTime()
	if !mtime.After(rd.last) && rd.rules != nil {
		return nil // no changes to rules file
	}

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("validator: error in opening %q: %v", file, err)
	}
	defer f.Close()

	var r []*Rule
	if err := json.NewDecoder(f).Decode(&r); err != nil {
		return fmt.Errorf("validator: error in JSON decoding rules file %q: %v", file, err)
	}

	rd.mu.Lock()
	rd.last = mtime
	rd.rules = r
	rd.mu.Unlock()
	return nil
}

// refreshRules refreshes the rules from the given file every
// 10 minutes.
func (rd *RequestDecoder) refreshRules(file string) {
	for {
		if err := rd.loadRules(file); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Minute * 10)
	}
}
