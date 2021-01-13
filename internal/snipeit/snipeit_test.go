// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package snipeit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/mock"

	client "github.com/influxdata/influxdb1-client/v2"
)

var (
	mux         *http.ServeMux // mux is the HTTP request multiplexer used with the test server.
	testBaseURL string
)

func TestStations(t *testing.T) {
	ic := &mock.InfluxClient{}
	ic.QueryFn = func(q client.Query) (*client.Response, error) {
		var resp *client.Response
		if err := json.Unmarshal([]byte(influxJSONResponse), &resp); err != nil {
			return nil, err
		}

		return resp, nil
	}

	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		w.Write(snipeITLocationJSON)
	})

	c, err := NewSnipeITService(testBaseURL, "testtoken", ic, "testdb")
	if err != nil {
		t.Fatalf("NewSnipeITService failed: %v", err)
	}
	ctx := context.Background()

	var testCases = []struct {
		in   *browser.Message
		want []string // Station ID's
	}{
		{
			&browser.Message{},
			[]string{"2", "3"},
		},
		{
			&browser.Message{
				Stations: []string{"2"},
			},
			[]string{"2"},
		},
		{
			&browser.Message{
				Stations: []string{"2", "3"},
			},
			[]string{"2", "3"},
		},
	}
	for _, tt := range testCases {
		t.Run(fmt.Sprintf("%v", tt.in), func(t *testing.T) {
			got, err := c.Stations(ctx, tt.in)
			if err != nil {
				t.Fatalf("Stations failed: %v", err)
			}

			if len(tt.want) != len(got) {
				t.Errorf("stations: got %d, want %d", len(got), len(tt.want))
			}

			for _, s := range got {
				if !inArray(s.ID, tt.want) {
					t.Errorf("station %q not found", s.ID)
				}
			}
		})
	}
}

func TestMain(m *testing.M) {
	mux = http.NewServeMux()

	// Run Mock SnipeIT API
	server := httptest.NewServer(mux)
	testBaseURL = server.URL

	// call flag.Parse() here if TestMain uses flags
	os.Exit(m.Run())
}

var influxJSONResponse = []byte(`{
	"results": [
		{
			"series": [
				{
					"name": "air_rh_avg",
					"columns": [
						"key",
						"value"
					],
					"values": [
						[
							"snipeit_location_ref",
							"2"
						],
						[
							"snipeit_location_ref",
							"3"
						]
					]
				}
			]
		},
		{
			"series": [
				{
					"name": "air_rh_std",
					"columns": [
						"key",
						"value"
					],
					"values": [
						[
							"snipeit_location_ref",
							"2"
						]
					]
				}
			]
		},
		{
			"series": [
				{
					"name": "air_rh_std",
					"columns": [
						"key",
						"value"
					],
					"values": [
						[
							"snipeit_location_ref",
							"2"
						]
					]
				}
			]
		}
	]
}
`)

var snipeITLocationJSON = []byte(`{
	"total": 20,
	"rows": [
		{
			"id": 1,
			"name": "LTER",
			"image": null,
			"address": null,
			"address2": null,
			"city": null,
			"state": null,
			"country": null,
			"zip": null,
			"assigned_assets_count": 0,
			"assets_count": 0,
			"users_count": 0,
			"currency": null,
			"created_at": {
				"datetime": "2019-06-21 08:09:57",
				"formatted": "2019-06-21 8:09AM"
			},
			"updated_at": {
				"datetime": "2019-06-21 08:09:57",
				"formatted": "2019-06-21 8:09AM"
			},
			"parent": null,
			"manager": null,
			"available_actions": {
				"update": false,
				"delete": false
			}
		},
		{
			"id": 2,
			"name": "P1",
			"image": "https://alpenv.assets.eurac.edu/uploads/locations/59-img-20160727-114714jpg.jpg",
			"address": "46.68586300000",
			"address2": "10.58294569000",
			"city": "P1/Raw",
			"state": "P1_2020.dat",
			"country": null,
			"zip": "1526",
			"assigned_assets_count": 0,
			"assets_count": 12,
			"users_count": 0,
			"currency": "pa",
			"created_at": {
				"datetime": "2019-05-03 11:10:43",
				"formatted": "2019-05-03 11:10AM"
			},
			"updated_at": {
				"datetime": "2020-01-07 11:38:40",
				"formatted": "2020-01-07 11:38AM"
			},
			"parent": {
				"id": 71,
				"name": "LTER"
			},
			"manager": null,
			"children": [

			],
			"available_actions": {
				"update": false,
				"delete": false
			}
		},
		{
			"id": 3,
			"name": "I1",
			"image": "https://alpenv.assets.eurac.edu/uploads/locations/60-img-20180524-163851-hdrjpg.jpg",
			"address": "46.68700895770",
			"address2": "10.57969320620",
			"city": "I1/Raw",
			"state": "I1_2020.dat",
			"country": null,
			"zip": "1490",
			"assigned_assets_count": 0,
			"assets_count": 13,
			"users_count": 0,
			"currency": "me",
			"created_at": {
				"datetime": "2019-05-03 11:10:43",
				"formatted": "2019-05-03 11:10AM"
			},
			"updated_at": {
				"datetime": "2020-01-07 11:39:49",
				"formatted": "2020-01-07 11:39AM"
			},
			"parent": {
				"id": 71,
				"name": "LTER"
			},
			"manager": null,
			"children": [

			],
			"available_actions": {
				"update": false,
				"delete": false
			}
		},
		{
			"id": 4,
			"name": "S3",
			"image": "https://alpenv.assets.eurac.edu/uploads/locations/3-img-20170913-174958jpg.jpg",
			"address": "46.76671000000",
			"address2": "10.71079000000",
			"city": "S3/Raw",
			"state": "S3_2020.dat",
			"country": null,
			"zip": "2680",
			"assigned_assets_count": 0,
			"assets_count": 27,
			"users_count": 0,
			"currency": "pa",
			"created_at": {
				"datetime": "2019-05-03 11:10:29",
				"formatted": "2019-05-03 11:10AM"
			},
			"updated_at": {
				"datetime": "2020-01-07 11:38:05",
				"formatted": "2020-01-07 11:38AM"
			},
			"parent": {
				"id": 71,
				"name": "LTER"
			},
			"manager": null,
			"children": [

			],
			"available_actions": {
				"update": false,
				"delete": false
			}
		}
	]
}
`)
