package ql

import "testing"

func TestWhereBuilder(t *testing.T) {
	testCases := []struct {
		in   Querier
		want string
	}{
		{Where(), ""},
		{Where(nil, And(), Eq(And(), "x", "a", "b")), "x='a' AND x='b'"},
		{Where(Eq(And(), "x", "a", "b")), "x='a' AND x='b'"},
		{Where(Eq(Or(), "x", "a", "b")), "x='a' OR x='b'"},
		{Where(Eq(Or(), "x", "a")), "x='a'"},
		{Where(Eq(Or(), "x", "")), ""},
		{Where(Eq(Or(), "x", "", "s", "")), "x='s'"},
		{Where(And(), Eq(And(), "a", "b")), "a='b'"},
		{Where(And(), Eq(Or(), "a", "b")), "a='b'"},
		{Where(Eq(Or(), "x", ""), And(), Eq(And(), "a", "b")), "a='b'"},
		{Where(Eq(Or(), "x", "a"), And(), Lte(And(), "y", "1")), "x='a' AND y<='1'"},
	}

	for _, tc := range testCases {
		got, _ := tc.in.Query()
		if got != tc.want {
			t.Errorf("got query %q, want %q", got, tc.want)
		}
	}
}

func TestShowTagValuesBuilder(t *testing.T) {
	testCases := []struct {
		in   Querier
		want string
	}{
		{ShowTagValues(), "SHOW TAG VALUES "},
		{ShowTagValues().From(), "SHOW TAG VALUES FROM /.*/"},
		{ShowTagValues().From("a"), "SHOW TAG VALUES FROM a"},
		{ShowTagValues().From("a", "b"), "SHOW TAG VALUES FROM a, b"},
		{ShowTagValues().From("a").WithKeyIn("b"), "SHOW TAG VALUES FROM a WITH KEY IN (\"b\")"},
		{ShowTagValues().From("a").WithKeyIn("b").Where(), "SHOW TAG VALUES FROM a WITH KEY IN (\"b\")"},
		{ShowTagValues().From("a").WithKeyIn("b").Where(Eq(And(), "x", "b")), "SHOW TAG VALUES FROM a WITH KEY IN (\"b\") WHERE x='b'"},
	}
	for _, tc := range testCases {
		if got, _ := tc.in.Query(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestSelectBuilder(t *testing.T) {
	testCases := []struct {
		in   Querier
		want string
	}{
		{Select(), "SELECT *"},
		{Select("a"), "SELECT a"},
		{Select("a").TZ("europe/rome"), "SELECT a TZ('europe/rome')"},
		{Select("a", "b"), "SELECT a, b"},
		{Select("a", "b").From("c"), "SELECT a, b FROM c"},
		{Select("a", "b").From("c").Where(Eq(And(), "x", "b")).GroupBy("t").OrderBy("a").ASC(), "SELECT a, b FROM c WHERE x='b' GROUP BY t ORDER BY a ASC"},
	}
	for _, tc := range testCases {
		if got, _ := tc.in.Query(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestShowMeasurementBuilder(t *testing.T) {
	testCases := []struct {
		in   Querier
		want string
	}{
		{ShowMeasurement(), "SHOW MEASUREMENTS"},
		{ShowMeasurement().With(EQ, "ab"), "SHOW MEASUREMENTS WITH MEASUREMENT = /ab/"},
		{ShowMeasurement().With("", ""), "SHOW MEASUREMENTS"},
		{ShowMeasurement().Where(), "SHOW MEASUREMENTS"},
		{ShowMeasurement().Where(Eq(And(), "a", "b")), "SHOW MEASUREMENTS WHERE a='b'"},
		{ShowMeasurement().With(EQ, "ab").Where(Eq(And(), "a", "b")), "SHOW MEASUREMENTS WITH MEASUREMENT = /ab/ WHERE a='b'"},
	}
	for _, tc := range testCases {
		if got, _ := tc.in.Query(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
