// Copyright 2020 Eurac Research. All rights reserved.

// Package http handles everything related to HTTP.
package http

import (
	"log"
	"net/http"

	"gitlab.inf.unibz.it/lter/browser"
)

const languageCookieName = "browser_lter_lang"

// ListenAndServe is a wrapper for http.ListenAndServe.
func ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

// Error writes an error message to the response.
func Error(w http.ResponseWriter, err error, code int) {
	// Log error.
	log.Printf("http error: %s (code=%d)", err, code)

	// Hide error message from client if it is internal or not found.
	if code == http.StatusInternalServerError || code == http.StatusNotFound {
		err = browser.ErrInternal
	}

	http.Error(w, err.Error(), code)
}

// grantAccess is a HTTP middlware function which grants access to the given
// handler that the requesting user is allowed to call the provided handler
// function. If not a http.NotFound will be returned.
func grantAccess(h http.HandlerFunc, roles ...browser.Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowed(r, roles...) {
			http.NotFound(w, r)
			return
		}

		h(w, r)
	}
}

// isAllowed checks if the current user makes part of the allowed roles.
func isAllowed(r *http.Request, roles ...browser.Role) bool {
	u := browser.UserFromContext(r.Context())

	for _, v := range roles {
		if u.Role == v {
			return true
		}
	}

	return false
}
