// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package http

import (
	"embed"
	"net/http"

	"github.com/euracresearch/browser"
)

var (
	//go:embed templates/* locale/*
	templateFS embed.FS

	//go:embed assets/*
	publicFS embed.FS
)

// Handler serves various HTTP endpoints.
type Handler struct {
	mux *http.ServeMux

	// analytics is a Google Analytics code.
	analytics string

	db             browser.Database
	stationService browser.StationService
}

// NewHandler creates a new HTTP handler with the given options and initializes
// all routes.
func NewHandler(options ...Option) *Handler {
	h := new(Handler)

	for _, option := range options {
		option(h)
	}

	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/", h.handleIndex())

	h.mux.HandleFunc("/en/hello/", h.handleHello())
	h.mux.HandleFunc("/it/hello/", h.handleHello())
	h.mux.HandleFunc("/de/hello/", h.handleHello())

	h.mux.HandleFunc("/en/", h.handleStaticPage())
	h.mux.HandleFunc("/it/", h.handleStaticPage())
	h.mux.HandleFunc("/de/", h.handleStaticPage())

	h.mux.HandleFunc("/l/", handleLanguage())

	h.mux.HandleFunc("/api/v1/stations/", h.handleStations())
	h.mux.HandleFunc("/api/v1/series", h.handleSeries())
	h.mux.HandleFunc("/api/v1/templates", grantAccess(h.handleCodeTemplate(), browser.FullAccess))

	h.mux.HandleFunc("robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/assets/robots.txt", http.StatusMovedPermanently)
	})

	// Setup endpoint to display deployed version.
	h.mux.HandleFunc("/debug/version", h.handleVersion)
	h.mux.HandleFunc("/debug/commit", h.handleCommit)

	h.mux.Handle("/assets/", http.FileServer(http.FS(publicFS)))

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

// WithStationService returns an option function for setting the handlers's
// stationService.
func WithStationService(s browser.StationService) Option {
	return func(h *Handler) {
		h.stationService = s
	}
}

// WithAnalyticsCode sets the Google Analytics code.
func WithAnalyticsCode(analytics string) Option {
	return func(h *Handler) {
		h.analytics = analytics
	}
}

func (h *Handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(browser.Version))
}

func (h *Handler) handleCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(browser.Commit))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
