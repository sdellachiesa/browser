// Copyright 2020 Eurac Research. All rights reserved.

package oauth2

import (
	"context"
	"errors"
	"fmt"

	"gitlab.inf.unibz.it/lter/browser"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	// MicrosoftIssuer is the ID token issure for microsoft personal accounts.
	MicrosoftIssuer = "https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"

	// ScientificNetIssuer is the ID token issure for ScientificNet accounts.
	ScientificNetIssuer = "https://login.microsoftonline.com/92513267-03e3-401a-80d4-c58ed6674e3b/v2.0"
)

// Guarantee we implement Provider.
var _ Provider = &Microsoft{}

// Microsoft is an OAuth2 provider for signing in using azure AD.
type Microsoft struct {
	Issuer      string
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

	provider, err := oidc.NewProvider(ctx, m.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oauth2(azure): error creating oidc provider: %v", err)
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: m.ClientID,
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
