// Copyright 2021 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package http

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/euracresearch/browser"
)

func (h *Handler) handleStations() http.HandlerFunc {
	funcMap := template.FuncMap{
		"T":  translate,
		"Is": isRole,
		"Mod": func(i int) bool {
			i++
			return (i % 2) == 0
		},
	}

	tmpl, err := template.New("station.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/station.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		station, err := h.stationService.Station(ctx, id)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		groups, err := h.db.GroupsByStation(ctx, id)
		if err != nil && !errors.Is(err, browser.ErrGroupsNotFound) {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Station  *browser.Station
			Groups   []browser.Group
			Language string
			User     *browser.User
		}{
			Station:  station,
			Groups:   groups,
			Language: languageFromCookie(r),
			User:     browser.UserFromContext(ctx),
		})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
		}

	}
}
