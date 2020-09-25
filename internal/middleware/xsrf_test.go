// Copyright 2020 Eurac Research. All rights reserved.

package middleware

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"golang.org/x/net/xsrftoken"
)

func TestXSRFProtect(t *testing.T) {
	const (
		testKey  = "You are not expected to understand this."
		testBody = `<input type="hidden" name="token" value="$$XSRFTOKEN$$">`
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testBody)
	})

	mw := XSRFProtect(testKey)
	ts := httptest.NewServer(mw(handler))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Errorf("GET returned error %v", err)
	}
	defer resp.Body.Close()

	gotBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	tokenRE := regexp.MustCompile(`value="(.*)"`)
	matches := tokenRE.FindStringSubmatch(string(gotBody))
	if matches == nil || len(matches) < 2 {
		t.Fatal("cannot extract nonce")
	}

	if !xsrftoken.Valid(matches[1], testKey, "", "") {
		t.Fatal("token is not valid")
	}

	res, err := ts.Client().PostForm(ts.URL, url.Values{"token": {matches[1]}})
	if err != nil {
		t.Errorf("POST returned error %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST: want status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	res, err = ts.Client().PostForm(ts.URL, url.Values{"token": {"randome"}})
	if err != nil {
		t.Errorf("POST returned error %v", err)
	}

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST: want status code %d, got %d", http.StatusForbidden, res.StatusCode)
	}

}
