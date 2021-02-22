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

		m, err := parseMessage(r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		ts, err := h.db.Series(ctx, m)
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

		m, err := parseMessage(r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		stmt := h.db.Query(ctx, m)

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

// parseForm parses form values from the given http.Request and returns a
// browser.Message. It performs basic validation for the given dates.
func parseMessage(r *http.Request) (*browser.Message, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	start, err := time.ParseInLocation("2006-01-02", r.FormValue("startDate"), browser.Location)
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.ParseInLocation("2006-01-02", r.FormValue("endDate"), browser.Location)
	if err != nil {
		return nil, fmt.Errorf("could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	if r.Form["measurements"] == nil {
		return nil, errors.New("at least one measurement must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("at least one station must be given")
	}

	return &browser.Message{
		Measurements: r.Form["measurements"],
		Stations:     r.Form["stations"],
		Landuse:      r.Form["landuse"],
		Start:        start,
		End:          end,
	}, nil
}
