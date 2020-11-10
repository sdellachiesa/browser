// Copyright 2020 Eurac Research. All rights reserved.

package http

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/middleware"
	"gitlab.inf.unibz.it/lter/browser/static"
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

	tmpl, err := static.ParseTemplates(template.New("base.tmpl").Funcs(funcMap), "html/base.tmpl", "html/index.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := browser.UserFromContext(ctx)

		// If the user is not public and has not signed the data usage
		// agreement, redirect it to sign it.
		if user.Role != browser.Public && !user.License {
			http.Redirect(w, r, "/hello", http.StatusTemporaryRedirect)
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
			languageFromCookie(r),
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

	tmpl, err := static.ParseTemplates(template.New("base.tmpl").Funcs(funcMap), "html/base.tmpl", "html/hello.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		lang := languageFromCookie(r)
		ctx := r.Context()
		user := browser.UserFromContext(ctx)

		const name = "license"
		license, err := static.File(filepath.Join("html", name, fmt.Sprintf("%s.%s.html", name, lang)))
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

	tmpl, err := static.ParseTemplates(template.New("base.tmpl").Funcs(funcMap), "html/base.tmpl", "html/page.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := browser.UserFromContext(ctx)
		lang := languageFromCookie(r)

		p := strings.TrimSuffix(r.URL.Path, "/")
		name := strings.TrimPrefix(p, fmt.Sprintf("/%s/", lang))
		filename := strings.ReplaceAll(name, "/", ".")

		// TODO: this is a special case for the info page only.
		if name == "info" && user.Role != browser.Public {
			filename = "internal.info"
		}

		p, err := static.File(filepath.Join("html", name, fmt.Sprintf("%s.%s.html", filename, lang)))
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

// TODO: extract to middleware?
func handleLanguage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := "en"

		switch r.URL.Path[len("/l/"):] {
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
			ref = refURL.Path
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

// translate is a template helper function for translating text in other languages.
func translate(key, lang string) template.HTML {
	j, err := static.File(filepath.Join("locale", fmt.Sprintf("%s.json", lang)))
	if err != nil {
		return template.HTML(key)
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		log.Printf("translation: %v\n", err)
		return template.HTML(key)
	}

	v, ok := m[key]
	if !ok {
		return template.HTML(key)
	}

	return template.HTML(v)
}
