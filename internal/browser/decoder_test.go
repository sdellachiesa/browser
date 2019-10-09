package browser

import (
	"reflect"
	"testing"
)

func TestInputFilter(t *testing.T) {
	testCases := []struct {
		index   int
		in      []string
		allowed []string
		want    []string
	}{
		{0, []string{}, []string{"a", "b"}, []string{"a", "b"}},
		{1, []string{"a"}, []string{"a", "b"}, []string{"a"}},
		{2, []string{"x"}, []string{"a", "b"}, []string{"a", "b"}},
		{3, []string{"x", "b"}, []string{"a", "b"}, []string{"b"}},
		{4, []string{}, []string{}, []string{}},
		{5, []string{"x", "y", "c"}, []string{"a", "b"}, []string{"a", "b"}},
		{6, []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{7, []string{"a", "b"}, []string{"a", "b", "c"}, []string{"a", "b"}},
		{8, []string{"b", "c"}, []string{"a", "b", "c"}, []string{"b", "c"}},
		{9, []string{"b", "c"}, []string{}, []string{"b", "c"}},
		{10, []string{"b'SELECT *", "c"}, []string{}, []string{"c"}},
		{11, []string{"b'SELECT *", "c"}, []string{"d"}, []string{"d"}},
		{12, []string{"b@", "c"}, []string{}, []string{"c"}},
		{13, []string{"b--SELECT *;", "c"}, []string{}, []string{"c"}},
	}

	rd := &RequestDecoder{}

	for _, tc := range testCases {
		got := rd.inputFilter(tc.in, tc.allowed)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%d: got %q, want %q", tc.index, got, tc.want)
		}
	}
}
