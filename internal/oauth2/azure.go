// Copyright 2020 Eurac Research. All rights reserved.

package oauth2

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"gitlab.inf.unibz.it/lter/browser"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	// Tenant is the Azure AD tenant.
	Tenant = "scientificnet.onmicrosoft.com"

	// Issuer is used for verifing the ID token.
	Issuer = "https://login.microsoftonline.com/92513267-03e3-401a-80d4-c58ed6674e3b/v2.0"
)

// Azure is a HTTP middleware for handling OAuth2 authentiction against
// Microsoft Azure AD.
type Azure struct {
	next     http.Handler
	config   *oauth2.Config
	state    string
	nonce    string
	verifier *oidc.IDTokenVerifier

	auth browser.Authenticator
}

// AzureOptions holds several options for the Azure OAuth2 autentication
// middleware.
type AzureOptions struct {
	ClientID    string
	Secret      string
	RedirectURL string
	State       string
	Nonce       string
}

// NewAzureOAuth2 returns a new Azure OAuth2 autentication middleware.
func NewAzureOAuth2(next http.Handler, auth browser.Authenticator, opts *AzureOptions) (*Azure, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, Issuer)
	if err != nil {
		return nil, fmt.Errorf("error creating oidc provider: %v", err)
	}

	a := &Azure{
		next: next,
		config: &oauth2.Config{
			ClientID:     opts.ClientID,
			ClientSecret: opts.Secret,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     microsoft.AzureADEndpoint(Tenant),
			RedirectURL:  opts.RedirectURL,
		},
		state: opts.State,
		nonce: opts.Nonce,
		verifier: provider.Verifier(&oidc.Config{
			ClientID: opts.ClientID,
		}),
		auth: auth,
	}

	return a, nil
}

func (a *Azure) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.URL.Path {
	default:
		u, err := a.auth.Validate(ctx, r)
		if err != nil {
			a.next.ServeHTTP(w, r)
			return
		}

		// Attach user information to the context of the request
		ctx := context.WithValue(ctx, browser.BrowserContextKey, u)
		a.next.ServeHTTP(w, r.WithContext(ctx))

	case "/auth/azure/logout":
		a.auth.Expire(w)

		redirect(w, r, "/", http.StatusSeeOther)

	case "/auth/azure/login":
		redirect(w, r, a.config.AuthCodeURL(a.state, oidc.Nonce(a.nonce)), http.StatusMovedPermanently)

	case "/auth/azure/callback":
		if r.URL.Query().Get("state") != a.state {
			msg := fmt.Sprintf("invalid state token, got %q, want %q.", r.FormValue("state"), a.state)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		token, err := a.config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			log.Println(err)
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}

		// Verify the ID Token signature and nonce.
		idToken, err := a.verifier.Verify(ctx, rawIDToken)
		if err != nil {
			log.Println(err)
			http.Error(w, "failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if idToken.Nonce != a.nonce {
			log.Println(err)
			http.Error(w, "invalid ID Token nonce", http.StatusInternalServerError)
			return
		}

		// Extract the roles claim.
		var claims struct {
			Username string   `json:"preferred_username"`
			Name     string   `json:"name"`
			Roles    []string `json:"roles"`
		}
		if err := idToken.Claims(&claims); err != nil {
			log.Println(err)
			http.Error(w, "error extracting claim from ID Token", http.StatusInternalServerError)
			return
		}

		u := &browser.User{
			Username: claims.Username,
			Name:     claims.Name,
			Role:     browser.Public,
		}

		if len(claims.Roles) >= 1 {
			u.Role = browser.NewRole(claims.Roles[0])
		}

		if err := a.auth.Authorize(ctx, w, u); err != nil {
			log.Println(err)
		}

		redirect(w, r, "/", http.StatusSeeOther)
	}

}

// redirect is a wrapper for http.Redirect
func redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")

	http.Redirect(w, r, url, code)
}
