//
// This file is adapted from augie.upspin.io/cmd/upspin-ui/static/gen.go.
//
// Package static provides access to static assets, such as HTML, CSS,
// JavaScript, and image files.
package static

import (
	"errors"
	"fmt"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"sync"
)

//go:generate go run makestatic.go

const pkgPath = "gitlab.inf.unibz.it/lter/browser/static"

var files map[string]string

var static struct {
	once sync.Once
	dir  string
}

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
	static.once.Do(func() {
		pkg, _ := build.Default.Import(pkgPath, "", build.FindOnly)
		if pkg == nil {
			return
		}
		static.dir = pkg.Dir
	})
	if static.dir == "" {
		return "", errors.New("could not find static assets")
	}
	b, err := ioutil.ReadFile(filepath.Join(static.dir, name))
	if err != nil {
		return "", err
	}
	return string(b), nil
}