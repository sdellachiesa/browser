package browser

import "testing"

func TestFilterQuery(t *testing.T) {
	testCases := []struct {
		opts *Filter
		want string
	}{
		{&Filter{}, "SHOW TAG VALUES FROM /.*/ WITH KEY IN (\"landuse\", \"snipeit_location_ref\")"},
		{
			&Filter{
				Stations: []string{"1"},
			},
			"SHOW TAG VALUES FROM /.*/ WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE snipeit_location_ref='1'",
		},
		{
			&Filter{
				Fields: []string{"a"},
			},
			"SHOW TAG VALUES FROM a WITH KEY IN (\"landuse\", \"snipeit_location_ref\")",
		},
		{
			&Filter{
				Landuse: []string{"pa"},
			},
			"SHOW TAG VALUES FROM /.*/ WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE landuse='pa'",
		},
		{
			&Filter{
				Stations: []string{"1"},
				Fields:   []string{"a"},
			},
			"SHOW TAG VALUES FROM a WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE snipeit_location_ref='1'",
		},
		{
			&Filter{
				Stations: []string{"1"},
				Landuse:  []string{"pa"},
			},
			"SHOW TAG VALUES FROM /.*/ WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE snipeit_location_ref='1' OR landuse='pa'",
		},
		{
			&Filter{
				Fields:  []string{"a"},
				Landuse: []string{"pa"},
			},
			"SHOW TAG VALUES FROM a WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE landuse='pa'",
		},
		{
			&Filter{
				Stations: []string{"1"},
				Fields:   []string{"a"},
				Landuse:  []string{"pa"},
			},
			"SHOW TAG VALUES FROM a WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE snipeit_location_ref='1' OR landuse='pa'",
		},
		{
			&Filter{
				Stations: []string{"1", "2"},
				Fields:   []string{"a", "b"},
				Landuse:  []string{"pa", "fo"},
			},
			"SHOW TAG VALUES FROM a,b WITH KEY IN (\"landuse\", \"snipeit_location_ref\") WHERE snipeit_location_ref='1' OR snipeit_location_ref='2' OR landuse='pa' OR landuse='fo'",
		},
	}

	for _, tc := range testCases {
		got, err := tc.opts.Query()
		if err != nil {
			t.Fatalf("FilterOptions.Query() returned an error: %v", err)
		}

		if got != tc.want {
			t.Errorf("FilterOptions.Query() returend %q, want %q", got, tc.want)
		}
	}
}
