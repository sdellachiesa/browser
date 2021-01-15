// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package http

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/encoding/csv"
	"github.com/euracresearch/browser/internal/encoding/csvf"
)

func (h *Handler) handleSeries() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
			return
		}

		f, err := browser.ParseSeriesFilterFromRequest(r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		ts, err := h.db.Series(ctx, f)
		if errors.Is(err, browser.ErrDataNotFound) {
			Error(w, err, http.StatusBadRequest)
			return
		}
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.csv", time.Now().Unix())
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)

		switch r.FormValue("format") {
		default:
			writer := csv.NewWriter(w)
			if err := writer.Write(ts); err != nil {
				Error(w, err, http.StatusInternalServerError)
			}

		case "wide":
			writer := csvf.NewWriter(w)
			if err := writer.Write(ts); err != nil {
				Error(w, err, http.StatusInternalServerError)
			}
		}
	}
}

func (h *Handler) handleCodeTemplate() http.HandlerFunc {
	var (
		tmpl struct {
			python, rlang *template.Template
		}
		err error
	)

	tmpl.python, err = template.ParseFS(templateFS, "templates/python.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	tmpl.rlang, err = template.ParseFS(templateFS, "templates/r.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
			return
		}

		var (
			t   *template.Template
			ext string
		)
		switch r.FormValue("language") {
		case "python":
			t = tmpl.python
			ext = "py"
		case "r":
			t = tmpl.rlang
			ext = "r"
		default:
			Error(w, browser.ErrInternal, http.StatusInternalServerError)
			return
		}

		f, err := browser.ParseSeriesFilterFromRequest(r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		stmt := h.db.Query(ctx, f)

		filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.%s", time.Now().Unix(), ext)
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		err = t.Execute(w, struct {
			Query    string
			Database string
		}{
			Query:    stmt.Query,
			Database: stmt.Database,
		})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
		}
	}
}
