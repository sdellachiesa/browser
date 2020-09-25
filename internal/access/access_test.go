// Copyright 2020 Eurac Research. All rights reserved.

package access

import (
	"context"
	"log"
	"reflect"
	"strings"
	"testing"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/mock"
)

// We test only Query method since all other public methods are the same and Query
// is the easiest to test.

// TestAccessFile tests the access control with a access file.
func TestAccessFile(t *testing.T) {
	db := &mock.Database{}
	db.QueryFn = func(ctx context.Context, m *browser.Message) *browser.Stmt {
		return &browser.Stmt{Query: messageString(t, m)}
	}

	a, err := New("testdata/access.json", db, nil)
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		in      *browser.Message
		role    browser.Role
		license bool
		want    string
	}{
		"PublicOK": {
			in: &browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{"S"},
			},
			role:    browser.Public,
			license: true,
			want:    "A-B-C_S",
		},
		"PublicRedactMeasurments": {
			in: &browser.Message{
				Measurements: []string{"A", "B", "C", "D"},
				Landuse:      []string{"L"},
			},
			role:    browser.Public,
			license: true,
			want:    "A-B-C_L",
		},
		"PublicEmtpy": {
			in: &browser.Message{
				Measurements: []string{},
				Stations:     []string{},
				Landuse:      []string{},
			},
			role:    browser.Public,
			license: true,
			want:    "A-B-C",
		},
		"PublicAll": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.Public,
			license: true,
			want:    "A-B-C_S_L",
		},
		"RoleNotFound": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.Role("knuth"),
			license: true,
			want:    "A-B-C_S_L",
		},
		"EmptyRole": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.Role(""),
			license: true,
			want:    "A-B-C_S_L",
		},
		"LicenseFalse": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.External,
			license: true,
			want:    "A-B-C_S_L",
		},
		"FullAccess": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.FullAccess,
			license: true,
			want:    "X_S_L",
		},
		"PublicNil": {
			in:      nil,
			role:    browser.Public,
			license: true,
			want:    "A-B-C",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			want := &browser.Stmt{
				Query: tc.want,
			}
			ctx := createContext(t, tc.role, tc.license)
			got := a.Query(ctx, tc.in)
			if got.Query != want.Query {
				log.Fatalf("(%s): got %s, want: %s", name, got.Query, want.Query)
			}

		})
	}
}

// TestAccessBuildInRules tests the access controls with no access file provided. So the build-in rules
// should be applied.
func TestAccessBuildInRules(t *testing.T) {
	db := &mock.Database{}
	db.QueryFn = func(ctx context.Context, m *browser.Message) *browser.Stmt {
		return &browser.Stmt{Query: messageString(t, m)}
	}

	a, err := New("", db, nil)
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		in      *browser.Message
		role    browser.Role
		license bool
		want    string
	}{
		"PublicOK": {
			in: &browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.Public,
			license: true,
			want:    "air_t_avg-air_rh_avg-wind_dir-wind_speed_avg-wind_speed_max-wind_speed-nr_up_sw_avg-precip_rt_nrt_tot-snow_height_S_L",
		},
		"PublicRedac": {
			in: &browser.Message{
				Measurements: []string{"air_t_avg", "sw_p"},
				Landuse:      []string{"L"},
			},
			role:    browser.Public,
			license: true,
			want:    "air_t_avg_L",
		},
		"PublicEmtpy": {
			in: &browser.Message{
				Measurements: []string{},
				Stations:     []string{},
				Landuse:      []string{},
			},
			role:    browser.Public,
			license: true,
			want:    "air_t_avg-air_rh_avg-wind_dir-wind_speed_avg-wind_speed_max-wind_speed-nr_up_sw_avg-precip_rt_nrt_tot-snow_height",
		},
		"LicesneFalse": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.External,
			license: false,
			want:    "air_t_avg-air_rh_avg-wind_dir-wind_speed_avg-wind_speed_max-wind_speed-nr_up_sw_avg-precip_rt_nrt_tot-snow_height_S_L",
		},
		"FullAccess": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.External,
			license: true,
			want:    "X_S_L",
		},
		"External": {
			in: &browser.Message{
				Measurements: []string{"X"},
				Stations:     []string{"S"},
				Landuse:      []string{"L"},
			},
			role:    browser.External,
			license: true,
			want:    "X_S_L",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			want := &browser.Stmt{
				Query: tc.want,
			}
			ctx := createContext(t, tc.role, tc.license)
			got := a.Query(ctx, tc.in)
			if got.Query != want.Query {
				log.Fatalf("(%s): got %s, want: %s", name, got.Query, want.Query)
			}

		})
	}
}

func TestClear(t *testing.T) {
	testCases := map[string]struct {
		in      []string
		allowed []string
		want    []string
	}{
		"0":  {[]string{}, []string{"a", "b"}, []string{"a", "b"}},
		"1":  {[]string{"a"}, []string{"a", "b"}, []string{"a"}},
		"2":  {[]string{"x"}, []string{"a", "b"}, []string{"a", "b"}},
		"3":  {[]string{"x", "b"}, []string{"a", "b"}, []string{"b"}},
		"4":  {[]string{}, []string{}, []string{}},
		"5":  {[]string{"x", "y", "c"}, []string{"a", "b"}, []string{"a", "b"}},
		"6":  {[]string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		"7":  {[]string{"a", "b"}, []string{"a", "b", "c"}, []string{"a", "b"}},
		"8":  {[]string{"b", "c"}, []string{"a", "b", "c"}, []string{"b", "c"}},
		"9":  {[]string{"b", "c"}, []string{}, []string{"b", "c"}},
		"10": {[]string{"b'SELECT *", "c"}, []string{}, []string{"c"}},
		"11": {[]string{"b'SELECT *", "c"}, []string{"d"}, []string{"d"}},
		"12": {[]string{"b@", "c"}, []string{}, []string{"c"}},
		"13": {[]string{"b--SELECT *;", "c"}, []string{}, []string{"c"}},
		"14": {[]string{"x"}, []string{"A", "b"}, []string{"A", "b"}},
		"15": {[]string{"B"}, []string{"a", "B"}, []string{"B"}},
	}

	a := &Access{}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			got := a.clear(tc.in, tc.allowed)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// messageString transforms the given browser.Message to a string so we can
// compare strings in a test output.
func messageString(t *testing.T, m *browser.Message) string {
	t.Helper()

	var s []string
	if len(m.Measurements) > 0 {
		s = append(s, strings.Join(m.Measurements, "-"))
	}
	if len(m.Stations) > 0 {
		s = append(s, strings.Join(m.Stations, "-"))
	}
	if len(m.Landuse) > 0 {
		s = append(s, strings.Join(m.Landuse, "-"))
	}

	return strings.Join(s, "_")
}

// createContext returns a new context with an browser.User embedded with the given
// role and license.
func createContext(t *testing.T, role browser.Role, lic bool) context.Context {
	t.Helper()

	u := &browser.User{
		Role:    role,
		License: lic,
	}
	return context.WithValue(context.Background(), browser.UserContextKey, u)
}
