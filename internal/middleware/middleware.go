// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package middleware implements a simple middleware pattern for HTTP handlers,
// along with implementation for some common middlewares.
package middleware

import (
	"net/http"
)

// A Middleware is a func that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain creates a new Middleware that applies a sequence of Middlewares, so
// that they execute in the given order when handling an http request.
//
// In other words, Chain(m1, m2)(handler) = m1(m2(handler))
// The call chain is from outer to inner.
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
