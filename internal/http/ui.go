// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/middleware"
)

func (h *Handler) handleIndex() http.HandlerFunc {
	funcMap := template.FuncMap{
		"T":         translate,
		"Is":        isRole,
		"HasSuffix": strings.HasSuffix,
		"Mod": func(i int) bool {
			i++
			return (i % 2) == 0
		},
		"Last": func(i, l int) bool {
			return i == (l - 1)
		},
	}

	tmpl, err := template.New("base.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/base.tmpl", "templates/index.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := browser.UserFromContext(ctx)
		lang := languageFromCookie(r)

		// If the user is not public and has not signed the data usage
		// agreement, redirect it to sign it.
		if user.Role != browser.Public && !user.License {
			http.Redirect(w, r, fmt.Sprintf("/%s/hello/", lang), http.StatusTemporaryRedirect)
			return
		}

		data, err := h.metadata.Stations(ctx, &browser.Message{})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Data          browser.Stations
			User          *browser.User
			Language      string
			Path          string
			AnalyticsCode string
			Token         string
			StartDate     string
			EndDate       string
		}{
			data,
			user,
			lang,
			r.URL.Path,
			h.analytics,
			middleware.XSRFTokenPlaceholder,
			time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
			time.Now().Format("2006-01-02"),
		})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
		}
	}
}

func (h *Handler) handleHello() http.HandlerFunc {
	funcMap := template.FuncMap{
		"T":  translate,
		"Is": isRole,
	}

	tmpl, err := template.New("base.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/base.tmpl", "templates/hello.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		lang := languageFromCookie(r)
		ctx := r.Context()
		user := browser.UserFromContext(ctx)

		const name = "license"
		license, err := templateFS.ReadFile(filepath.Join("templates", name, fmt.Sprintf("%s.%s.html", name, lang)))
		if err != nil {
			Error(w, err, http.StatusNotFound)
			return
		}

		data, err := h.metadata.Stations(ctx, &browser.Message{})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Data          browser.Stations
			User          *browser.User
			Language      string
			Path          string
			AnalyticsCode string
			Token         string
			Content       template.HTML
		}{
			data,
			user,
			lang,
			name,
			h.analytics,
			middleware.XSRFTokenPlaceholder,
			template.HTML(license),
		})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
		}
	}
}

func (h *Handler) handleStaticPage() http.HandlerFunc {
	funcMap := template.FuncMap{
		"T":  translate,
		"Is": isRole,
	}

	tmpl, err := template.New("base.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/base.tmpl", "templates/page.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := browser.UserFromContext(ctx)
		lang := languageFromCookie(r)

		name, err := pageNameFromPath(r.URL.Path)
		if err != nil {
			// On error we assume a language changes is wanted.
			p := fmt.Sprintf("/l%s", r.URL.Path)
			http.Redirect(w, r, p, http.StatusTemporaryRedirect)
			return
		}
		filename := fmt.Sprintf("%s.%s.html", strings.ReplaceAll(name, "/", "."), lang)

		// TODO: this is a special case for the info page only.
		if name == "info" && user.Role != browser.Public {
			filename = fmt.Sprintf("internal.info.%s.html", lang)
		}

		p, err := templateFS.ReadFile(filepath.Join("templates", name, filename))
		if err != nil {
			Error(w, err, http.StatusNotFound)
			return
		}

		data, err := h.metadata.Stations(ctx, &browser.Message{})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, struct {
			Data          browser.Stations
			User          *browser.User
			Language      string
			Path          string
			AnalyticsCode string
			Content       template.HTML
		}{
			data,
			user,
			lang,
			name,
			h.analytics,
			template.HTML(p),
		})
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
		}
	}
}

// pageNameFromPath is a helper for extracing the page name from the request
// URL. It assumes that the page name is always in the URL.
func pageNameFromPath(p string) (string, error) {
	p = strings.TrimSuffix(strings.TrimPrefix(p, "/"), "/")
	names := strings.Split(p, "/")

	// There must be at least two values inside names, the language and the page
	// name. Otherwise we assume something is wrong.
	if len(names) < 2 {
		return "", errors.New("no name found in url path")
	}
	return names[len(names)-1], nil
}

// TODO: extract to middleware?
func handleLanguage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := "en"

		p := strings.TrimSuffix(r.URL.Path, "/")
		switch p[len("/l/"):] {
		case "de":
			l = "de"
		case "it":
			l = "it"
		}

		http.SetCookie(w, &http.Cookie{
			Name:  languageCookieName,
			Value: l,
			Path:  "/",
		})

		ref := "/"
		refURL, err := url.Parse(r.Referer())
		if err == nil && refURL.Path != "" {
			// The language part has to be replaced with the actual language in
			// the referer.
			name, err := pageNameFromPath(refURL.Path)
			if err != nil {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
			ref = fmt.Sprintf("/%s/%s", l, name)
		}

		w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
		w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Accel-Expires", "0")

		http.Redirect(w, r, ref, http.StatusSeeOther)
	}
}

// languageFromCookie reads the language settings from a cookie.
func languageFromCookie(r *http.Request) string {
	c, err := r.Cookie(languageCookieName)
	if err != nil {
		return "en"
	}
	return c.Value
}

// isRole is a template helper function for verifying a users role.
func isRole(r browser.Role, s string) bool {
	return r == browser.NewRole(s)
}

// translate is a template helper function for translating text to other
// languages.
func translate(key, lang string) template.HTML {
	j, err := templateFS.ReadFile(filepath.Join("locale", fmt.Sprintf("%s.json", lang)))
	if err != nil {
		log.Println(err)
		return template.HTML(key)
	}

	var m map[string]string
	if err := json.Unmarshal(j, &m); err != nil {
		log.Printf("translation: %v\n", err)
		return template.HTML(key)
	}

	v, ok := m[key]
	if !ok {
		return template.HTML(key)
	}

	return template.HTML(v)
}
