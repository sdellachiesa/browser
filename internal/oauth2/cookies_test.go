// Copyright 2019 Eurac Research. All rights reserved.

package oauth2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.inf.unibz.it/lter/browser"

	"github.com/google/go-cmp/cmp"
)

func TestAuthorize(t *testing.T) {
	testCases := map[string]struct {
		in  *browser.User
		err error
	}{
		"nil":   {nil, browser.ErrAuthentication},
		"empty": {&browser.User{}, nil},
		"public": {
			&browser.User{
				Name:     "test",
				Username: "testusername",
				Role:     browser.DefaultRole,
			},
			nil,
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			c := &Cookie{"testsecret"}
			w := httptest.NewRecorder()

			err := c.Authorize(context.Background(), w, tc.in)
			if err != nil {
				if errors.Is(err, tc.err) {
					return
				}
				t.Fatalf("Authorize: returned error: %v", err)
			}

			cookies := w.Header().Get("Set-Cookie")

			if cookies == "" {
				t.Fatal("Expected some cookies but got zero")
			}

			if !strings.Contains(cookies, fmt.Sprintf("%s=", DefaultCookieName)) {
				t.Errorf("no cookie found with name %s", DefaultCookieName)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		name       string
		signingKey string
		in         *browser.User
		want       *browser.User
		err        error
	}{
		"nil": {
			DefaultCookieName,
			"testkey",
			nil,
			nil,
			ErrTokenInvalid,
		},
		"empty": {
			DefaultCookieName,
			"testkey",
			&browser.User{},
			&browser.User{Role: browser.DefaultRole},
			nil,
		},
		"name": {
			"testcookie",
			"testkey",
			&browser.User{},
			nil,
			http.ErrNoCookie,
		},
		"full": {
			DefaultCookieName,
			"testkey",
			&browser.User{
				Name:     "test",
				Username: "testusername",
				Role:     browser.FullAccess,
			},
			&browser.User{
				Name:     "test",
				Username: "testusername",
				Role:     browser.FullAccess,
			},
			nil,
		},
		"partial": {
			DefaultCookieName,
			"testkey",
			&browser.User{
				Name:     "A",
				Username: "a@b.com",
			},
			&browser.User{
				Name:     "A",
				Username: "a@b.com",
				Role:     browser.DefaultRole,
			},
			nil,
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			c := &Cookie{"testkey"}
			token, _ := c.newJWT(tc.in)

			req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
			req.AddCookie(&http.Cookie{
				Name:  tc.name,
				Value: token,
			})

			got, err := c.Validate(context.Background(), req)
			if err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatal(err)
				}
			}

			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
