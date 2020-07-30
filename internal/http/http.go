// Copyright 2020 Eurac Research. All rights reserved.

// Package http handles everything related to HTTP.
package http

import (
	"log"
	"net/http"

	"gitlab.inf.unibz.it/lter/browser"
)

// A Middleware is a func that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain creates a new Middleware that applies a sequence of Middlewares, so
// that they execute in the given order when handling an http request.
//
// In other words, Chain(m1, m2)(handler) = m1(m2(handler))
//
// A similar pattern is used in e.g. github.com/justinas/alice:
// https://github.com/justinas/alice/blob/ce87934/chain.go#L45
//
// Taken from:
// https://github.com/golang/pkgsite/blob/master/internal/middleware/middleware.go#L21
func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := range middlewares {
			h = middlewares[len(middlewares)-1-i](h)
		}
		return h
	}
}

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

// SecureHeaders adds security-related headers to all responses.
func SecureHeaders() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Don't allow frame embedding.
			w.Header().Set("X-Frame-Options", "deny")
			// Prevent MIME sniffing.
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Block cross-site scripting attacks.
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			h.ServeHTTP(w, r)
		})
	}
}
