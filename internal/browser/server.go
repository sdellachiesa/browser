// Copyright 2019 Eurac Research. All rights reserved.
package browser

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	text "text/template"
	"time"

	"golang.org/x/net/xsrftoken"
	"golang.org/x/text/language"
	"gopkg.in/russross/blackfriday.v2"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/static"
)

const langCookieName = "browser-lang"

// Backend is an interface for retrieving LTER data.
type Backend interface {
	Get(auth.Role) Stations
	Series(context.Context, *request) ([][]string, error)
	Query(context.Context, *request) string
}

// Server represents an HTTP server for serving the LTER Browser
// application.
type Server struct {
	// key to prevent request forgery; static for server's lifetime.
	key string
	mux *http.ServeMux

	html struct {
		index, page *template.Template
	}

	text struct {
		python, rlang *text.Template
	}

	// Influx database name used inside code templates.
	database string

	db      Backend
	matcher language.Matcher

	// Google Analytics code
	analyticsCode string
}

// NewServer initializes and returns a new HTTP server. It takes
// one or more option function and applies them in order to the
// server.
func NewServer(options ...Option) (*Server, error) {
	s := &Server{
		mux: http.NewServeMux(),
	}

	for _, option := range options {
		option(s)
	}

	if err := s.parseTemplate(); err != nil {
		return nil, fmt.Errorf("parsing templates: %v", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("must provide and option func that specifies a Backend")
	}

	key, err := generateKey()
	if err != nil {
		return nil, err
	}
	s.key = key

	s.matcher = language.NewMatcher([]language.Tag{
		language.English, // The first language is used as fallback.
		language.Italian,
		language.German,
	})

	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/p/", s.handlePage)
	s.mux.HandleFunc("/l/", s.handleLanguage)
	s.mux.HandleFunc("/static/", static.ServeContent)

	s.mux.HandleFunc("/api/v1/series", s.handleSeries)
	s.mux.HandleFunc("/api/v1/template", grantAccessTo(s.handleTemplate, auth.FullAccess))

	return s, nil
}

// Option controls some aspects of the server.
type Option func(*Server)

// WithBackend returns an option function for setting
// the server's backend.
func WithBackend(b Backend) Option {
	return func(s *Server) {
		s.db = b
	}
}

// WithDatabase returns an options function for setting
// the server's database used insde the code templates.
func WithInfluxDB(db string) Option {
	return func(s *Server) {
		s.database = db
	}
}

func WithAnalyticsCode(analytics string) Option {
	return func(s *Server) {
		s.analyticsCode = analytics
	}
}

func (s *Server) parseTemplate() error {
	funcMap := template.FuncMap{
		"T":         s.translate,
		"HasSuffix": strings.HasSuffix,
		"Last": func(i, l int) bool {
			return i == (l - 1)
		},
	}

	var err error
	s.html.index, err = static.ParseTemplates(template.New("base.tmpl").Funcs(funcMap), "base.tmpl", "nav.tmpl")
	if err != nil {
		return err
	}

	s.html.page, err = static.ParseTemplates(template.New("base.tmpl").Funcs(funcMap), "base.tmpl", "page.tmpl", "nav.tmpl")
	if err != nil {
		return err
	}

	s.text.python, err = static.ParseTextTemplates(nil, "python.tmpl")
	if err != nil {
		return err
	}

	s.text.rlang, err = static.ParseTextTemplates(nil, "r.tmpl")
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {

	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	err := s.html.index.Execute(w, struct {
		Data            Stations
		StartDate       string
		EndDate         string
		IsAuthenticated bool
		Role            auth.Role
		Language        string
		Path            string
		Token           string
		AnalyticsCode   string
	}{
		s.db.Get(role),
		time.Now().AddDate(0, -6, 0).Format("2006-01-02"),
		time.Now().Format("2006-01-02"),
		auth.IsAuthenticated(r),
		role,
		s.language(r),
		r.URL.Path,
		xsrftoken.Generate(s.key, "", "/api/v1/"),
		s.analyticsCode,
	})
	if err != nil {
		err = fmt.Errorf("handleIndex: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	if filepath.Ext(r.URL.Path) != ".md" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}

	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		role = auth.Public
	}

	lang := s.language(r)
	p, err := static.File(filepath.Join("pages", lang, filepath.Base(r.URL.Path)))
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	if err := s.html.page.Execute(w, struct {
		IsAuthenticated bool
		Role            auth.Role
		Language        string
		Content         interface{}
		Path            string
		AnalyticsCode   string
	}{
		auth.IsAuthenticated(r),
		role,
		lang,
		template.HTML(blackfriday.Run([]byte(p))),
		r.URL.Path,
		s.analyticsCode,
	}); err != nil {
		err = fmt.Errorf("handleIndex: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

func (s *Server) handleLanguage(w http.ResponseWriter, r *http.Request) {
	l := r.URL.Path[len("/l/"):]

	if validLanguage(l) {
		http.SetCookie(w, &http.Cookie{
			Name:  langCookieName,
			Value: l,
			Path:  "/",
		})
	}

	ref := "/"
	u, err := url.Parse(r.Referer())
	if err == nil {
		ref = u.Path
	}

	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")

	http.Redirect(w, r, ref, http.StatusSeeOther)
}

// request represents an request received from the client.
type request struct {
	measurements, stations, landuse []string
	start                           time.Time
	end                             time.Time
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	if !xsrftoken.Valid(r.FormValue("token"), s.key, "", "/api/v1/") {
		http.Error(w, "Invalid XSRF token", http.StatusForbidden)
		return
	}

	req, err := parseForm(r)
	if err != nil {
		err = fmt.Errorf("handleSeries: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	ctx := r.Context()
	b, err := s.db.Series(ctx, req)
	if err != nil {
		err = fmt.Errorf("handleSeries: %v", err)
		reportError(w, r, err)
		return
	}

	filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.csv", time.Now().Unix())
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	csv.NewWriter(w).WriteAll(b)
}

func (s *Server) handleTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
		return
	}

	req, err := parseForm(r)
	if err != nil {
		err = fmt.Errorf("handleTemplate: error in decoding or validating data: %v", err)
		reportError(w, r, err)
		return
	}

	var (
		tmpl *text.Template
		ext  string
	)
	switch r.FormValue("language") {
	case "python":
		tmpl = s.text.python
		ext = "py"
	case "r":
		tmpl = s.text.rlang
		ext = "r"
	default:
		reportError(w, r, errors.New("language not supported"))
		return
	}

	filename := fmt.Sprintf("LTSER_IT25_Matsch_Mazia_%d.%s", time.Now().Unix(), ext)
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	ctx := r.Context()
	q := s.db.Query(ctx, req)

	err = tmpl.Execute(w, struct {
		Query    string
		Database string
	}{
		Query:    q,
		Database: s.database,
	})
	if err != nil {
		err = fmt.Errorf("handleTemplate: error in executing template: %v", err)
		reportError(w, r, err)
		return
	}
}

func (s *Server) translate(key, lang string) string {
	j, err := static.File(filepath.Join("locale", fmt.Sprintf("%s.json", lang)))
	if err != nil {
		return key
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		log.Printf("translation: %v\n", err)
		return key
	}

	v, ok := m[key]
	if !ok {
		return key
	}

	return v
}

func (s *Server) language(r *http.Request) string {
	tag, _ := language.MatchStrings(s.matcher, langFromCookie(r), r.Header.Get("Accept-Language"))

	if !validLanguage(tag.String()) {
		return "en"
	}

	return tag.String()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set default security headers.
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "deny")

	s.mux.ServeHTTP(w, r)
}

func langFromCookie(r *http.Request) string {
	c, err := r.Cookie(langCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func validLanguage(s string) bool {
	switch s {
	case "en":
		return true
	case "de":
		return true
	case "it":
		return true
	default:
		return false
	}
}

// parseForm parses form values from the given http.Request and returns
// an request. It performs basic validation for end date and download
// limit.  Data in influx is UTC but LTER data is UTC+1 therefor
// parseForm will adapt start and end times. It will shift the start
// time to -1 hour and will set the end time to 22:59:59 in order to
// capture a full day.
func parseForm(r *http.Request) (*request, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	start, err := time.Parse("2006-01-02", r.FormValue("startDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.Parse("2006-01-02", r.FormValue("endDate"))
	if err != nil {
		return nil, fmt.Errorf("could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	// Limit download of data to one year
	limit := time.Date(end.Year()-1, end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if start.Before(limit) {
		return nil, errors.New("time range is greater then a year")
	}

	if r.Form["measurements"] == nil {
		return nil, errors.New("at least one measurement must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("at least one station must be given")
	}

	return &request{
		measurements: r.Form["measurements"],
		stations:     r.Form["stations"],
		landuse:      r.Form["landuse"],
		start:        start.Add(-1 * time.Hour),
		end:          time.Date(end.Year(), end.Month(), end.Day(), 22, 59, 59, 59, time.UTC),
	}, nil
}

func grantAccessTo(h http.HandlerFunc, roles ...auth.Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowed(r, roles...) {
			http.NotFound(w, r)
			return
		}

		h(w, r)
	}
}

func isAllowed(r *http.Request, roles ...auth.Role) bool {
	role, ok := r.Context().Value(auth.JWTClaimsContextKey).(auth.Role)
	if !ok {
		return false
	}

	for _, v := range roles {
		if role == v {
			return true
		}
	}

	return false
}

func reportError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%v\n", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func generateKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
