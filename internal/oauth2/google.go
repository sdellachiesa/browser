// Copyright 2020 Eurac Research. All rights reserved.

package oauth2

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc"
	"gitlab.inf.unibz.it/lter/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Guarantee we implement Provider.
var _ Provider = &Google{}

// Google is an OAuth2 provider for signing in using Google accounts.
type Google struct {
	ClientID    string
	Secret      string
	RedirectURL string
	Nonce       string
}

// Name returns the name of the provider.
func (g *Google) Name() string {
	return "google"
}

// Config is the Google OAuth2 configuration.
func (g *Google) Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.Secret,
		Endpoint:     google.Endpoint,
		RedirectURL:  g.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}
}

// User returns an browser.User with information from Google.
func (g *Google) User(ctx context.Context, token *oauth2.Token) (*browser.User, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	const issuer = "https://accounts.google.com"
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oauth2(google): error creating oidc provider: %v", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: g.Config().ClientID,
	})

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != g.Nonce {
		return nil, errors.New("nonce in id token is not right")
	}

	u := new(browser.User)
	if err := idToken.Claims(&u); err != nil {
		return nil, err
	}
	u.Role = browser.External
	u.Provider = g.Name()

	return u, nil
}
