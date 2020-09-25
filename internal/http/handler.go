// Copyright 2020 Eurac Research. All rights reserved.

package http

import (
	"net/http"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/static"
)

// Handler serves various HTTP endpoints.
type Handler struct {
	mux *http.ServeMux

	// analytics is a Google Analytics code.
	analytics string

	db       browser.Database
	metadata browser.Metadata
}

func NewHandler(options ...Option) *Handler {
	h := new(Handler)

	for _, option := range options {
		option(h)
	}

	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/", h.handleIndex())
	h.mux.HandleFunc("/agreement", h.handleDataLicenseAgreement())

	h.mux.HandleFunc("/en/", h.handleStaticPage())
	h.mux.HandleFunc("/it/", h.handleStaticPage())
	h.mux.HandleFunc("/de/", h.handleStaticPage())

	h.mux.HandleFunc("/l/", handleLanguage())
	h.mux.HandleFunc("/static/", static.ServeContent)

	h.mux.HandleFunc("/api/v1/series", h.handleSeries())
	h.mux.HandleFunc("/api/v1/templates", grantAccess(h.handleCodeTemplate(), browser.FullAccess))

	return h
}

// Option controls some aspects of the Handler.
type Option func(h *Handler)

// WithDatabase returns an options function for setting the handler's database
// backend.
func WithDatabase(db browser.Database) Option {
	return func(h *Handler) {
		h.db = db
	}
}

// WithMetadata returns an option function for setting the handlers's metadata
// backend.
func WithMetadata(m browser.Metadata) Option {
	return func(h *Handler) {
		h.metadata = m
	}
}

// WithAnalyticsCode sets the Google Analytics code.
func WithAnalyticsCode(analytics string) Option {
	return func(h *Handler) {
		h.analytics = analytics
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
