// This file is adapted from augie.upspin.io/cmd/upspin-ui/static/gen.go.
//
// Package static provides access to static assets, such as HTML, CSS,
// JavaScript, and image files.
package static

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:generate go run makestatic.go

var files map[string]string

// File returns the file rooted at "gitlab.inf.unibz.it/lter/lter/internal/static" either
// from an in-memory map or, if no map was generated, the contents of the file
// from disk.
func File(name string) (string, error) {
	if files != nil {
		b, ok := files[name]
		if !ok {
			return "", fmt.Errorf("file not found '%v'", name)
		}
		return b, nil

	}
	b, err := ioutil.ReadFile(filepath.Join("static", name))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path[1:]
		if p == "" {
			p = "index.html"
		}

		b, err := File(p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, path.Base(p), time.Now(), strings.NewReader(b))
	})
}
