// Copyright 2019 Eurac Research. All rights reserved.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Azure handles the authentication with the Azure AD by requesting an ID Token
// and verifing it.
func Azure(next http.Handler, cfg *oauth2.Config, jwtKey []byte) http.Handler {
	state := randomString(32)
	appNonce := randomString(32)

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/92513267-03e3-401a-80d4-c58ed6674e3b/v2.0")
	if err != nil {
		log.Fatalf("error creating oidc provider: %v", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := "Public"

		switch r.URL.Path {
		case "/auth/azure/logout":
			if _, err := NewJWT(jwtKey, role, w); err != nil {
				http.Error(w, "auth: error in creating JWT token: "+err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
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

			if len(claims.Roles) >= 1 {
				role = claims.Roles[0]
			}

			// Generate JWT and set cookie.
			if _, err := NewJWT(jwtKey, role, w); err != nil {
				http.Error(w, "auth: error in creating JWT token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/", http.StatusMovedPermanently)
		}

		// Get the JWT if not found a new one is created.
		claims, err := GetJWTClaims(jwtKey, w, r)
		if err != nil {
			http.Error(w, "auth: error with JWT token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), JWTClaimsContextKey, claims.Group)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func randomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(b)
}
