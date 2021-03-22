// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package influx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/mock"
	"github.com/google/go-cmp/cmp"
	client "github.com/influxdata/influxdb1-client/v2"
)

var (
	// testSelectQuery is the query we expect in test to lookup an user.
	testSelectQuery = "select updated from test where email='jane@example.com' and provider='test' group by provider,fullname,email,picture,license,role"

	// testDeleteQuery is the query we expect to get when delete an user.
	testDeleteQuery = "delete from test where email='jane@example.com' and provider='test' and time=1603116509454279000"
)

func TestGet(t *testing.T) {
	testCases := map[string]struct {
		in   *browser.User
		err  error
		want *browser.User
	}{
		"nil": {
			in:   nil,
			err:  browser.ErrUserNotFound,
			want: nil,
		},
		"ok": {
			in: &browser.User{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Provider: "test",
			},
			err: nil,
			want: &browser.User{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				License:  true,
				Picture:  "/static/images/jane.png",
				Provider: "test",
				Role:     browser.External,
			},
		},
		"partial": {
			in: &browser.User{
				Email: "jane@example.com",
			},
			err:  browser.ErrUserNotFound,
			want: nil,
		},
		"notfound": {
			in: &browser.User{
				Name:     "John Doe",
				Email:    "John@example.com",
				Provider: "test",
			},
			err:  browser.ErrUserNotFound,
			want: nil,
		},
	}

	us := &UserService{
		Client: &mock.InfluxClient{
			QueryFn: userQueryFnHelper(t),
		},
		Database: "testdb",
		Env:      "test",
	}
	ctx := context.Background()

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			got, err := us.Get(ctx, tc.in)
			if err != tc.err {
				t.Fatal(err)
			}

			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testCases := map[string]struct {
		in   *browser.User
		want error
	}{
		"nil": {
			in:   nil,
			want: browser.ErrUserNotValid,
		},
		"ok": {
			in: &browser.User{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Provider: "test",
			},
			want: nil,
		},
		"partial": {
			in: &browser.User{
				Email: "jane@example.com",
			},
			want: browser.ErrUserNotValid,
		},
		"notfound": {
			in: &browser.User{
				Name:     "John Doe",
				Email:    "John@example.com",
				Provider: "test",
			},
			want: browser.ErrUserNotFound,
		},
	}

	us := &UserService{
		Client: &mock.InfluxClient{
			QueryFn: userQueryFnHelper(t),
		},
		Database: "testdb",
		Env:      "test",
	}
	ctx := context.Background()

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			got := us.Delete(ctx, tc.in)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

		})
	}

}

func TestUpdate(t *testing.T) {
	testCases := map[string]struct {
		in   *browser.User
		want error
	}{
		"nil": {
			in:   nil,
			want: browser.ErrUserNotValid,
		},
		"ok": {
			in: &browser.User{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Provider: "test",
				Role:     browser.Public,
			},
			want: nil,
		},
		"partial": {
			in: &browser.User{
				Email: "jane@example.com",
			},
			want: browser.ErrUserNotValid,
		},
		"notfound": {
			in: &browser.User{
				Name:     "John Doe",
				Email:    "John@example.com",
				Provider: "test",
			},
			want: browser.ErrUserNotFound,
		},
	}

	us := &UserService{
		Client: &mock.InfluxClient{
			QueryFn: userQueryFnHelper(t),
			WriteFn: userWriteFnHelper(t),
		},
		Database: "testdb",
		Env:      "test",
	}
	ctx := context.Background()

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			got := us.Update(ctx, tc.in)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

		})
	}

}

func TestCreate(t *testing.T) {
	testCases := map[string]struct {
		in   *browser.User
		want error
	}{
		"nil": {
			in:   nil,
			want: browser.ErrUserNotValid,
		},
		"already": {
			in: &browser.User{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Provider: "test",
				Role:     browser.Public,
			},
			want: browser.ErrUserAlreadyExists,
		},
		"partial": {
			in: &browser.User{
				Email: "jane@example.com",
			},
			want: browser.ErrUserNotValid,
		},
	}

	us := &UserService{
		Client: &mock.InfluxClient{
			WriteFn: userWriteFnHelper(t),
			QueryFn: userQueryFnHelper(t),
		},
		Database: "testdb",
		Env:      "test",
	}
	ctx := context.Background()

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			got := us.Create(ctx, tc.in)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}

		})
	}

}
func userWriteFnHelper(t *testing.T) func(bp client.BatchPoints) error {
	t.Helper()

	return func(bp client.BatchPoints) error {
		if len(bp.Points()) != 1 {
			return errors.New("error")
		}
		p := bp.Points()[0]

		if p.Name() != "test" {
			return errors.New("error")
		}
		want := map[string]string{
			"provider": "test",
			"fullname": "Jane Doe",
			"email":    "jane@example.com",
			"license":  "false",
			"role":     string(browser.Public),
		}
		diff := cmp.Diff(want, p.Tags())
		if diff != "" {
			return fmt.Errorf("mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

// userQueryFnHelper is a helper which returns a QueryFn needed for the
// mock.InfluxClient. Depending on the input query it will to a select or delete
// operation. More over it will validate the input query with the global defined
// test queries.
func userQueryFnHelper(t *testing.T) func(q client.Query) (*client.Response, error) {
	t.Helper()
	return func(q client.Query) (*client.Response, error) {
		inQuery := strings.ToLower(q.Command)

		switch {
		case strings.HasPrefix(inQuery, "select"):
			if testSelectQuery != inQuery {
				return &client.Response{}, nil
			}

			f, err := os.Open(filepath.Join("testdata", "users.json"))
			if err != nil {
				return nil, err
			}
			defer f.Close()

			dec := json.NewDecoder(f)
			dec.UseNumber()

			var resp *client.Response
			if err := dec.Decode(&resp); err != nil {
				return nil, err
			}

			return resp, nil

		case strings.HasPrefix(inQuery, "delete"):
			if testDeleteQuery != inQuery {
				return nil, errors.New("something went wrong")
			}

			return &client.Response{}, nil
		}

		return nil, errors.New("unexpected error")
	}
}
