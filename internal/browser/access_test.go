package browser

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
)

func TestEnforce(t *testing.T) {
	testCases := []struct {
		index   string
		in      []string
		allowed []string
		want    []string
	}{
		{"0", []string{}, []string{"a", "b"}, []string{"a", "b"}},
		{"1", []string{"a"}, []string{"a", "b"}, []string{"a"}},
		{"2", []string{"x"}, []string{"a", "b"}, []string{"a", "b"}},
		{"3", []string{"x", "b"}, []string{"a", "b"}, []string{"b"}},
		{"4", []string{}, []string{}, []string{}},
		{"5", []string{"x", "y", "c"}, []string{"a", "b"}, []string{"a", "b"}},
		{"6", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"7", []string{"a", "b"}, []string{"a", "b", "c"}, []string{"a", "b"}},
		{"8", []string{"b", "c"}, []string{"a", "b", "c"}, []string{"b", "c"}},
		{"9", []string{"b", "c"}, []string{}, []string{"b", "c"}},
		{"10", []string{"b'SELECT *", "c"}, []string{}, []string{"c"}},
		{"11", []string{"b'SELECT *", "c"}, []string{"d"}, []string{"d"}},
		{"12", []string{"b@", "c"}, []string{}, []string{"c"}},
		{"13", []string{"b--SELECT *;", "c"}, []string{}, []string{"c"}},
		{"14", []string{"x"}, []string{"A", "b"}, []string{"A", "b"}},
		{"14", []string{"B"}, []string{"a", "B"}, []string{"B"}},
	}

	a := &Access{}

	for _, tc := range testCases {
		t.Run(tc.index, func(t *testing.T) {
			got := a.enforce(tc.in, tc.allowed)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	a := ParseAccessFile("testdata/access.json")

	testCases := []struct {
		ctx  context.Context
		in   *request
		want *request
	}{
		{
			ctxWithRole("Public"),
			&request{},
			&request{
				measurements: []string{"A", "B", "C"},
				stations:     []string{},
				landuse:      []string{},
			},
		},
		{
			ctxWithRole("Public"),
			&request{
				measurements: []string{},
				stations:     []string{},
				landuse:      []string{},
			},
			&request{
				measurements: []string{"A", "B", "C"},
				stations:     []string{},
				landuse:      []string{},
			},
		},
		{
			ctxWithRole("Public"),
			&request{
				measurements: []string{"D"},
				stations:     []string{},
				landuse:      []string{},
			},
			&request{
				measurements: []string{"A", "B", "C"},
				stations:     []string{},
				landuse:      []string{},
			},
		},
		{
			ctxWithRole("Public"),
			&request{
				measurements: []string{"D"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
			&request{
				measurements: []string{"A", "B", "C"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
		},
		{
			ctxWithRole("Public"),
			&request{
				measurements: []string{"C"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
			&request{
				measurements: []string{"C"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
		},
		{
			ctxWithRole("Internal"),
			&request{
				measurements: []string{"C"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
			&request{
				measurements: []string{"A", "b"},
				stations:     []string{"1"},
				landuse:      []string{"me"},
			},
		},
		{
			ctxWithRole("Internal"),
			&request{
				measurements: []string{"b"},
				stations:     []string{"1", "2"},
				landuse:      []string{},
			},
			&request{
				measurements: []string{"b"},
				stations:     []string{"1"},
				landuse:      []string{},
			},
		},
		{
			ctxWithRole("External"),
			&request{
				measurements: []string{"A"},
				stations:     []string{"1", "2"},
				landuse:      []string{},
			},
			&request{
				measurements: []string{"A"},
				stations:     []string{"2"},
				landuse:      []string{"me", "ma"},
			},
		},
		{
			ctxWithRole("External"),
			&request{
				measurements: []string{"A"},
				stations:     []string{"1", "2"},
				landuse:      []string{"boh"},
			},
			&request{
				measurements: []string{"A"},
				stations:     []string{"2"},
				landuse:      []string{"me", "ma"},
			},
		},
		{
			ctxWithRole("External"),
			&request{
				measurements: []string{},
				stations:     []string{"2"},
				landuse:      []string{"me"},
			},
			&request{
				measurements: []string{"A"},
				stations:     []string{"2"},
				landuse:      []string{"me"},
			},
		},
		{
			ctxWithRole("WrongACLKey"),
			&request{
				measurements: []string{},
				stations:     []string{"2"},
				landuse:      []string{"me"},
			},
			&request{
				measurements: []string{
					"air_t_avg",
					"air_rh_avg",
					"wind_dir",
					"wind_speed_avg",
					"wind_speed_max",
					"wind_speed",
					"nr_up_sw_avg",
					"precip_rt_nrt_tot",
					"snow_height",
				},
				stations: []string{"2"},
				landuse:  []string{"me"},
			},
		},
	}

	for _, tc := range testCases {
		a.Filter(tc.ctx, tc.in)

		if !reflect.DeepEqual(tc.in, tc.want) {
			t.Errorf("got %v, want %v", tc.in, tc.want)
		}
	}
}

func ctxWithRole(name string) context.Context {
	return context.WithValue(context.Background(), auth.JWTClaimsContextKey, auth.Role(name))
}

func TestRule(t *testing.T) {
	a := ParseAccessFile("testdata/access.json")

	testCases := []struct {
		in   string
		want *Rule
	}{
		{"Public", &Rule{
			Name: "Public",
			ACL: &AccessControlList{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{},
				Landuse:      []string{},
			},
		}},
		{"FullAccess", &Rule{
			Name: "FullAccess",
			ACL: &AccessControlList{
				Measurements: []string{},
				Stations:     []string{},
				Landuse:      []string{},
			},
		}},
		{"WrongNameKey", defaultRule},
		{"WrongACLKey", defaultRule},
		{"Missing", defaultRule},
		{"", defaultRule},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("get Rule: %s", tc.in), func(t *testing.T) {
			got := a.Rule(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNames(t *testing.T) {
	a := ParseAccessFile("testdata/access.json")

	want := []string{"Public", "FullAccess", "Internal", "External", "Empty"}
	got := a.Names()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
