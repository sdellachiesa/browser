// Copyright 2019 Eurac Research. All rights reserved.
package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2"
)

// Azure returns a new http.Handler for handling authentication
// with the Azure AD.
// On default it will read the role claim from an stored JWT token
// and pass it to the next handler as context value.
// On the Oauth2 flow it will request an ID Token from Azure AD,
// verify it and read the role claim. Store it in a newly created JWT
// token and redirect to /.
func Azure(next http.Handler, cfg *oauth2.Config, state, appNonce string, jwtKey []byte) http.Handler {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/92513267-03e3-401a-80d4-c58ed6674e3b/v2.0")
	if err != nil {
		log.Fatalf("error creating oidc provider: %v", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/azure/logout":
			http.SetCookie(w, &http.Cookie{
				Name:    jwtCookieName,
				Value:   "",
				Path:    "/",
				Expires: time.Now().Add(-time.Hour * 24),
			})
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return

		case "/auth/azure":
			http.Redirect(w, r, cfg.AuthCodeURL(state, oidc.Nonce(appNonce)), http.StatusMovedPermanently)
			return

		case "/auth/azure/callback":
			if r.URL.Query().Get("state") != state {
				msg := fmt.Sprintf("invalid state token, got %q, want %q.", r.FormValue("state"), state)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			token, err := cfg.Exchange(ctx, r.URL.Query().Get("code"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			rawIDToken, ok := token.Extra("id_token").(string)
			if !ok {
				http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
				return
			}

			// Verify the ID Token signature and nonce.
			idToken, err := verifier.Verify(ctx, rawIDToken)
			if err != nil {
				http.Error(w, "failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if idToken.Nonce != appNonce {
				http.Error(w, "invalid ID Token nonce", http.StatusInternalServerError)
				return
			}

			// Extract the roles claim.
			var claims struct {
				Roles []string `json:"roles"`
			}
			if err := idToken.Claims(&claims); err != nil {
				http.Error(w, "error extracting claim from ID Token", http.StatusInternalServerError)
				return
			}

			role := Public
			if len(claims.Roles) >= 1 {
				role = ParseRole(claims.Roles[0])
			}

			// Generate JWT and set cookie.
			if _, err := NewJWT(jwtKey, role, w); err != nil {
				http.Error(w, "auth: error in creating JWT token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return

		default:
			// Get the role fromt the JWT
			role, err := RoleFromJWT(jwtKey, r)
			if err == http.ErrNoCookie { // TODO: In Go 1.13 use errors.Is()
				err = nil
			}
			if ve, ok := err.(*jwt.ValidationError); ok {
				log.Printf("auth: JWT validation error: %v", ve)
				http.Redirect(w, r, "/auth/azure/logout", http.StatusMovedPermanently)
				return
			}
			if err != nil {
				log.Printf("auth: error with JWT token: %v", err)
				http.Error(w, "auth: "+http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), JWTClaimsContextKey, role)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
			return
		}
	})
}
