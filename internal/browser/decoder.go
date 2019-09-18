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
	"sync"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/ql"
)

type Decoder interface {
	DecodeAndValidate(r *http.Request) (ql.Querier, error)
}

type RequestDecoder struct {
	mu    sync.RWMutex // guards the fields below
	last  time.Time
	rules []*Rule
}

func NewRequestDecoder(file string) *RequestDecoder {
	rd := &RequestDecoder{}
	if err := rd.loadRules(file); err != nil {
		log.Fatal(err)
	}
	go rd.refreshRules(file)

	return rd
}

// TODO: Validation of identifiers
func (rd *RequestDecoder) DecodeAndValidate(r *http.Request) (ql.Querier, error) {
	rule, err := rd.Rule(r.Context())
	if err != nil {
		return nil, err
	}

	var f *Filter
	switch r.Header.Get("content-type") {
	case "application/x-www-form-urlencoded": // FORM Submit
		f, err = rd.decodeForm(r)
		if err != nil {
			return nil, err
		}

		f.qType = SeriesQuery
	default: // JSON
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			if err == io.EOF {
				return rule.Policy, nil
			}
			return nil, err
		}
		defer r.Body.Close()

		f.qType = UpdateQuery
	}

	f.Fields = Allowed(f.Fields, rule.Policy.Fields)
	f.Stations = Allowed(f.Stations, rule.Policy.Stations)
	f.Landuse = Allowed(f.Landuse, rule.Policy.Landuse)

	return f, nil
}

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

func Allowed(in []string, acl []string) []string {
	if len(in) < 1 {
		return acl
	}
	if len(acl) < 1 {
		return in
	}

	m := make(map[string]struct{}, len(in))
	for _, v := range in {
		m[v] = struct{}{}
	}

	var c []string
	for _, v := range acl {
		if _, found := m[v]; found {
			c = append(c, v)
		}
	}

	return c
}

type Rule struct {
	Role   string
	Policy *Filter
}

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

func (rd *RequestDecoder) refreshRules(file string) {
	for {
		if err := rd.loadRules(file); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Minute * 1)
	}
}
