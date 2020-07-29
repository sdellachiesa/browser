// Copyright 2020 Eurac Research. All rights reserved.

package http

import (
	"log"
	"net/http"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser/static"
)

const RobotsFilePathName = "/robots.txt"

// Robots is a HTTP middleware which serves the given filename as "/robots.txt".
func Robots(filename string) Middleware {
	s, err := static.File(filename)
	if err != nil {
		log.Fatalf("robots.txt: %v", err)
	}

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == RobotsFilePathName {
				http.ServeContent(w, r, RobotsFilePathName, time.Now(), strings.NewReader(s))
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
