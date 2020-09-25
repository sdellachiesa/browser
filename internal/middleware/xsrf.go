// Copyright 2020 Eurac Research. All rights reserved.

package middleware

import (
	"bytes"
	"log"
	"net/http"

	"gitlab.inf.unibz.it/lter/browser"
	"golang.org/x/net/xsrftoken"
)

// XSRFTokenPlaceholder should be used as the value for XSRF in rendered content.
// It is substituted for the actual token value by the XSRFProtect middleware.
const XSRFTokenPlaceholder = "$$XSRFTOKEN$$"

// XSRFProtect is a HTTP middlware adding XSRF/CSRF token protection for non-safe HTTP Methods.
func XSRFProtect(key string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isSafeMethod(r.Method) {
				if !xsrftoken.Valid(r.FormValue("token"), key, "", "") {
					http.Error(w, browser.ErrInvalidToken.Error(), http.StatusForbidden)
					return
				}
			}

			crw := &capturingResponseWriter{ResponseWriter: w}
			h.ServeHTTP(crw, r)
			body := bytes.ReplaceAll(crw.bytes(), []byte(XSRFTokenPlaceholder), []byte(xsrftoken.Generate(key, "", "")))
			if _, err := w.Write(body); err != nil {
				log.Printf("XSRFProtect, writing: %v", err)
			}
		})
	}
}

// capturingResponseWriter is an http.ResponseWriter that captures
// the body for later processing.
type capturingResponseWriter struct {
	http.ResponseWriter
	buf bytes.Buffer
}

func (c *capturingResponseWriter) Write(b []byte) (int, error) {
	return c.buf.Write(b)
}

func (c *capturingResponseWriter) bytes() []byte {
	return c.buf.Bytes()
}

// isSafeMethod checks if the given method is considered safe. Safe
// methods are  GET/HEAD/OPTIONS/TRACE.
func isSafeMethod(m string) bool {
	switch m {
	default:
		return false
	case http.MethodGet:
		return true
	case http.MethodHead:
		return true
	case http.MethodOptions:
		return true
	case http.MethodTrace:
		return true
	}

}
