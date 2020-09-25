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
	"github.com/gorilla/securecookie"
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
				Name: "test",
				Role: browser.DefaultRole,
			},
			nil,
		},
	}

	for k, tc := range testCases {
		t.Run(k, func(t *testing.T) {
			c := &Cookie{
				Secret: "testsecret",
				Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
			}
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

func TestValidateNilUser(t *testing.T) {
	c := &Cookie{
		Secret: "testsecret",
		Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
	}

	req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: "test",
	})

	_, err := c.Validate(context.Background(), req)
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestValidateEmptyUser(t *testing.T) {
	in := &browser.User{}
	want := &browser.User{Role: browser.DefaultRole}

	c := &Cookie{
		Secret: "testsecret",
		Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
	}

	token, _ := c.newJWT(in)
	encoded, err := c.Cookie.Encode(DefaultCookieName, token)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: encoded,
	})

	got, _ := c.Validate(context.Background(), req)

	diff := cmp.Diff(want, got)
	if diff != "" {
		t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateWrongCookieName(t *testing.T) {
	in := &browser.User{}
	want := http.ErrNoCookie

	c := &Cookie{
		Secret: "testsecret",
		Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
	}

	token, _ := c.newJWT(in)
	encoded, err := c.Cookie.Encode("no", token)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
	req.AddCookie(&http.Cookie{
		Name:  "no",
		Value: encoded,
	})

	_, err = c.Validate(context.Background(), req)
	if !errors.Is(err, want) {
		t.Fatalf("expected error %v, got %v", want, err)
	}
}

func TestValidateOK(t *testing.T) {
	in := &browser.User{
		Name: "test",
		Role: browser.FullAccess,
	}
	want := &browser.User{
		Name: "test",
		Role: browser.FullAccess,
	}

	c := &Cookie{
		Secret: "testsecret",
		Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
	}

	token, _ := c.newJWT(in)
	encoded, err := c.Cookie.Encode(DefaultCookieName, token)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: encoded,
	})

	got, _ := c.Validate(context.Background(), req)

	diff := cmp.Diff(want, got)
	if diff != "" {
		t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
	}
}

func TestValidatePartialUser(t *testing.T) {
	in := &browser.User{
		Name: "test",
	}
	want := &browser.User{
		Name: "test",
		Role: browser.Public,
	}

	c := &Cookie{
		Secret: "testsecret",
		Cookie: securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
	}

	token, _ := c.newJWT(in)
	encoded, err := c.Cookie.Encode(DefaultCookieName, token)
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("", "https://browser.lter.eurac.edu", nil)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCookieName,
		Value: encoded,
	})

	got, _ := c.Validate(context.Background(), req)

	diff := cmp.Diff(want, got)
	if diff != "" {
		t.Fatalf("Validate() mismatch (-want +got):\n%s", diff)
	}
}
