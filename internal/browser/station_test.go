package browser

import (
	"reflect"
	"testing"
)

func TestGet(t *testing.T) {
	stations := Stations{&Station{ID: "a"}, &Station{ID: "b"}}

	testCases := []struct {
		in   string
		want *Station
		ok   bool
	}{
		{"a", &Station{ID: "a"}, true},
		{"", nil, false},
		{"c", nil, false},
	}

	for _, tc := range testCases {
		got, ok := stations.Get(tc.in)

		if ok != tc.ok {
			t.Errorf("got %v, want %v", ok, tc.ok)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got %v, want %v", got, tc.want)
		}
	}
}

func TestAppendIfMissing(t *testing.T) {
	testCases := []struct {
		in    string
		slice []string
		want  []string
	}{
		{"a", []string{}, []string{"a"}},
		{"", []string{}, []string{""}},
		{"", []string{"a"}, []string{"a", ""}},
		{"b", []string{"a", "b"}, []string{"a", "b"}},
		{"C", []string{"a", "b"}, []string{"a", "b", "C"}},
	}

	for _, tc := range testCases {
		got := appendIfMissing(tc.slice, tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got %v, want %v", got, tc.want)
		}
	}

}
