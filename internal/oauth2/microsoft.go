// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc"
	"gitlab.inf.unibz.it/lter/browser"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// Guarantee we implement Provider.
var _ Provider = &Microsoft{}

// Microsoft is an OAuth2 provider for signing in using azure AD.
type Microsoft struct {
	Provider    string
	ClientID    string
	Secret      string
	RedirectURL string
	Nonce       string
}

// Name returns the name of provider.
func (m *Microsoft) Name() string {
	return m.Provider
}

// Config is the Microsoft OAuth2 configuration.
func (m *Microsoft) Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     m.ClientID,
		ClientSecret: m.Secret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     microsoft.AzureADEndpoint(""),
		RedirectURL:  m.RedirectURL,
	}
}

// User returns an browser.User with information from Azure AD.
func (m *Microsoft) User(ctx context.Context, token *oauth2.Token) (*browser.User, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	keySet := oidc.NewRemoteKeySet(ctx, "https://login.microsoftonline.com/common/discovery/v2.0/keys")
	verifier := oidc.NewVerifier("https://login.microsoftonline.com/common/v2.0", keySet, &oidc.Config{
		ClientID: m.ClientID,

		// TODO: don't know how to fix this since logins from other
		// tenants will have different issuer.
		SkipIssuerCheck: true,
	})

	// Verify the ID Token signature and nonce.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != m.Nonce {
		return nil, errors.New("nonce in id token is not right")
	}

	// Extract the roles claim.
	var claims struct {
		Username string   `json:"preferred_username"`
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Roles    []string `json:"roles"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}

	u := &browser.User{
		Name:     claims.Name,
		Email:    claims.Email,
		Provider: m.Name(),
		Role:     browser.External,
	}

	if len(claims.Roles) >= 1 {
		u.Role = browser.NewRole(claims.Roles[0])
	}

	return u, nil
}
