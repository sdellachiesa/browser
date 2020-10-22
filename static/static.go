// Copyright 2019 Eurac Research. All rights reserved.

// Package static provides access to static assets, such as HTML, CSS,
// JavaScript, and image files.
// This file is adapted from augie.upspin.io/cmd/upspin-ui/static/gen.go.
package static

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	text "text/template"
)

//go:generate go run static_gen.go

var (
	// Exclude defines a slice of file extensions which will
	// not be served by the ServeContent function.
	Exclude = []string{".tmpl"}

	files map[string]string
)

var static struct {
	once sync.Once
	dir  string
}

// File returns the file rooted at "gitlab.inf.unibz.it/lter/browser/static"
// either from an in-memory map or, if no map was generated,
// the contents of the file from disk.
func File(name string) (string, error) {
	if files != nil {
		b, ok := files[name]
		if !ok {
			// If the asset is not found in the in memory structure try to look
			// at file system level.
			data, err := ioutil.ReadFile(filepath.Join("static", name))
			if err != nil {
				return "", fmt.Errorf("file not found '%v'", name)
			}
			b = string(data)
		}
		return b, nil
	}

	static.once.Do(func() {
		b, err := run("go", "list", "-f", "{{.Dir}}", "gitlab.inf.unibz.it/lter/browser/static")
		if err != nil {
			return
		}
		static.dir = strings.Trim(b, "\n")
	})

	if static.dir == "" {
		return "", errors.New("static dir not found")
	}

	b, err := ioutil.ReadFile(filepath.Join(static.dir, name))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func run(n string, args ...string) (string, error) {
	c := exec.Command(n, args...)
	bb := &bytes.Buffer{}
	c.Stdout = bb
	c.Stderr = bb
	err := c.Run()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, bb)
	}

	return bb.String(), nil
}

// ServeContent serves static content for HTTP servers.
// Files matching any file extension defined in Exclude will
// be excluded from serving.
func ServeContent(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path[1:], "static/")

	for _, ext := range Exclude {
		if ext == filepath.Ext(p) {
			http.NotFound(w, r)
			return
		}
	}

	b, err := File(p)
	if err != nil {
		http.Error(w, "static asset not found: "+err.Error(), http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, path.Base(p), time.Now(), strings.NewReader(b))
}

// ParseTemplates creates a new "html/template" from the given files.
// If t is not nil it will be used as base template.
// It is a copy of the parseFiles function of:
// https://github.com/golang/go/blob/master/src/html/template/template.go#L404
// It reads files from the internal files map instead from disk directly.
func ParseTemplates(t *template.Template, filenames ...string) (*template.Template, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("static: no files named in call to ParseTemplates")
	}

	for _, filename := range filenames {
		s, err := File(filename)
		if err != nil {
			return nil, err
		}

		name := filepath.Base(filename)
		// First template becomes return value if not already defined,
		// and we use that one for subsequent New calls to associate
		// all the templates together. Also, if this file has the same name
		// as t, this file becomes the contents of t, so
		//  t, err := New(name).Funcs(xxx).ParseFiles(name)
		// works. Otherwise we create a new template associated with t.
		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// ParseTextTemplates is equivalent to ParseTemplate only that it
// returns parsed "text/template".
func ParseTextTemplates(t *text.Template, filenames ...string) (*text.Template, error) {
	if len(filenames) == 0 {
		return nil, fmt.Errorf("static: no files named in call to ParseTemplates")
	}

	for _, filename := range filenames {
		s, err := File(filename)
		if err != nil {
			return nil, err
		}

		name := filepath.Base(filename)
		var tmpl *text.Template
		if t == nil {
			t = text.New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}
