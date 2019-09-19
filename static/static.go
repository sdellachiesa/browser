// This file is adapted from augie.upspin.io/cmd/upspin-ui/static/gen.go.
//
// Package static provides access to static assets, such as HTML, CSS,
// JavaScript, and image files.
package static

import (
	"fmt"
	"io/ioutil"
	"strings"
)

//go:generate go run makestatic.go

var files map[string]string

// File returns the file rooted at "gitlab.inf.unibz.it/lter/browser/static" either
// from an in-memory map or, if no map was generated, the contents of the file
// from disk.
func File(name string) (string, error) {
	if files != nil {
		name = strings.TrimPrefix(name, "static/")
		b, ok := files[name]
		if !ok {
			return "", fmt.Errorf("file not found '%v'", name)
		}
		return b, nil

	}

	b, err := ioutil.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
