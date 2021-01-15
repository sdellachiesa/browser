// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package oauth2 provides an handler for handling OAuth2 authentication flows
// and the implementation of several OAuth2 providers.
package oauth2

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/euracresearch/browser"
	"golang.org/x/oauth2"
)

// Provider are the common parameters all OAuth2 providers should implement.
type Provider interface {
	// Name returns the name of the provider.
	Name() string
	// Config returns the OAuth2 config of the provider.
	Config() *oauth2.Config
	// User returns user information from the provider.
	User(context.Context, *oauth2.Token) (*browser.User, error)
}

// Authenticator represents a service for authenticating users.
type Authenticator interface {
	// Validate returns an authenticated User if a valid user session is found.
	Validate(context.Context, *http.Request) (*browser.User, error)

	// Authorize will create a new user session for authenticated users.
	Authorize(context.Context, http.ResponseWriter, *browser.User) error

	// Expire will logout the authenticated User.
	Expire(http.ResponseWriter)
}

// Handler handles OAuth2 authorization flows and different account aspects.
type Handler struct {
	Next  http.Handler
	State string
	Nonce string
	Auth  Authenticator
	Users browser.UserService

	mux *http.ServeMux
}

// Register registers all the routes for the given provider.
func (h *Handler) Register(p Provider) {
	if h.mux == nil {
		h.mux = http.NewServeMux()
		h.mux.HandleFunc("/auth/account/license", h.license())
		//h.mux.HandleFunc("/auth/account/cancel", h.cancel())
	}

	h.mux.HandleFunc("/auth/"+p.Name()+"/login", h.login(p.Config()))
	h.mux.HandleFunc("/auth/"+p.Name()+"/callback", h.callback(p))
	h.mux.HandleFunc("/auth/"+p.Name()+"/logout", h.logout())
}

func (h *Handler) login(config *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(h.State, oidc.Nonce(h.Nonce)), http.StatusTemporaryRedirect)
	}
}

func (h *Handler) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.Auth.Expire(w)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func (h *Handler) callback(p Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != h.State {
			log.Printf("oauth2(%s): invalid state token, got %q, want %q", p.Name(), r.FormValue("state"), h.State)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		ctx := r.Context()
		token, err := p.Config().Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			log.Printf("oauth2(%s): error in exchange: %v\n", p.Name(), err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		u, err := p.User(ctx, token)
		if err != nil {
			log.Printf("oauth2(%s): error in retriving user: %v\n", p.Name(), err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		if !u.Valid() {
			msg := "error user not valid missing 'name' or 'email'"
			log.Printf("oauth2(%s): %s %v\n", p.Name(), msg, u)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		// Check if the user is already registered. If not create a new user.
		user, err := h.Users.Get(ctx, u)
		if errors.Is(err, browser.ErrUserNotFound) {
			err = h.Users.Create(ctx, u)
			user = u
		}
		if err != nil {
			log.Printf("oauth2(%s): error getting user: %v\n", p.Name(), err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		if err := h.Auth.Authorize(ctx, w, user); err != nil {
			log.Printf("oauth2(%s): error in authorizing user: %v\n", p.Name(), err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

	}
}

func (h *Handler) license() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Expected POST request", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		user, err := h.Auth.Validate(ctx, r)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		// License already signed.
		if user.License {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		switch r.FormValue("agreement") {
		case "1":
			user.License = true
			if err := h.Users.Update(ctx, user); err != nil {
				log.Println(err)
			}
			if err := h.Auth.Authorize(ctx, w, user); err != nil {
				log.Println(err)
			}
		default:
			if err := h.Users.Delete(ctx, user); err != nil {
				log.Println(err)
			}
			h.Auth.Expire(w)
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

// TODO: for now disabled, maybe we will introduce a new super admin role which
// has the right do delete account data using the web interface. Currently this
// must be done manually.
//func (h *Handler) cancel() http.HandlerFunc {
//  return func(w http.ResponseWriter, r *http.Request) {
//      ctx := r.Context()
//      user, err := h.Auth.Validate(ctx, r)
//      if err != nil {
//          log.Printf("oauth2: cancel: validation failed: %v\n", err)
//          http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
//          return
//      }
//      if err := h.Users.Delete(ctx, user); err != nil {
//          log.Printf("oauth2: cancel: error in deleting user: %v\n", err)
//          http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
//          return
//      }
//
//      h.Auth.Expire(w)
//
//      http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
//  }
//}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch {
	default:
		u, err := h.Auth.Validate(ctx, r)
		if err != nil {
			h.Next.ServeHTTP(w, r)
			return
		}

		// Attach user information to the context of the request
		ctx = context.WithValue(ctx, browser.UserContextKey, u)
		h.Next.ServeHTTP(w, r.WithContext(ctx))

	case strings.HasPrefix(r.URL.Path, "/auth"):
		h.mux.ServeHTTP(w, r)
	}

}
