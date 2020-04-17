// Copyright 2020 Eurac Research. All rights reserved.

package access

import (
	"context"
	"reflect"
	"testing"

	"gitlab.inf.unibz.it/lter/browser"
)

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

func TestRedact(t *testing.T) {
	a, err := New("testdata/access.json", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		ctx  context.Context
		in   *browser.Message
		want *browser.Message
	}{
		{
			withCTX(browser.Public),
			&browser.Message{},
			&browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{},
				Landuse:      []string{},
			},
		},
		{
			withCTX(browser.Public),
			&browser.Message{
				Measurements: []string{},
				Stations:     []string{},
				Landuse:      []string{},
			},
			&browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{},
				Landuse:      []string{},
			},
		},
		{
			withCTX(browser.Public),
			&browser.Message{
				Measurements: []string{"D"},
				Stations:     []string{},
				Landuse:      []string{},
			},
			&browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{},
				Landuse:      []string{},
			},
		},
		{
			withCTX(browser.Public),
			&browser.Message{
				Measurements: []string{"D"},
				Stations:     []string{"1"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: []string{"A", "B", "C"},
				Stations:     []string{"1"},
				Landuse:      []string{"me"},
			},
		},
		{
			withCTX(browser.Public),
			&browser.Message{
				Measurements: []string{"C"},
				Stations:     []string{"1"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: []string{"C"},
				Stations:     []string{"1"},
				Landuse:      []string{"me"},
			},
		},
		{
			withCTX("WrongACLKey"),
			&browser.Message{
				Measurements: []string{},
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: defaultRule.ACL.Measurements,
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
		},
		{
			withCTX(""),
			&browser.Message{
				Measurements: []string{},
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: defaultRule.ACL.Measurements,
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
		},
		{
			withCTX(browser.Role("")),
			&browser.Message{
				Measurements: []string{},
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: defaultRule.ACL.Measurements,
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
		},
		{
			withCTX(browser.Role("knuth")),
			&browser.Message{
				Measurements: []string{},
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
			&browser.Message{
				Measurements: defaultRule.ACL.Measurements,
				Stations:     []string{"2"},
				Landuse:      []string{"me"},
			},
		},
	}

	for _, tc := range testCases {
		a.redact(tc.ctx, tc.in)

		if !reflect.DeepEqual(tc.in, tc.want) {
			t.Errorf("got %v, want %v", tc.in, tc.want)
		}
	}
}

func withCTX(role browser.Role) context.Context {
	u := &browser.User{Role: role}
	return context.WithValue(context.Background(), browser.BrowserContextKey, u)
}
