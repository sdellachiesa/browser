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
	// Tenant is the Azure AD tenant.
	Tenant = "scientificnet.onmicrosoft.com"

	// Issuer is used for verifying the ID token.
	Issuer = "https://login.microsoftonline.com/92513267-03e3-401a-80d4-c58ed6674e3b/v2.0"
)

// Guarantee we implement Provider.
var _ Provider = &Azure{}

// Azure is an OAuth2 provider for signing in using azure AD.
type Azure struct {
	ClientID    string
	Secret      string
	RedirectURL string
	Nonce       string
}

// Name returns the name of provider.
func (s *Azure) Name() string {
	return "azure"
}

// Config is the Azure OAuth2 configuration.
func (a *Azure) Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.ClientID,
		ClientSecret: a.Secret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     microsoft.AzureADEndpoint(Tenant),
		RedirectURL:  a.RedirectURL,
	}
}

// User returns an browser.User with information from Azure AD.
func (a *Azure) User(ctx context.Context, token *oauth2.Token) (*browser.User, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	provider, err := oidc.NewProvider(ctx, Issuer)
	if err != nil {
		return nil, fmt.Errorf("oauth2(azure): error creating oidc provider: %v", err)
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: a.ClientID,
	})

	// Verify the ID Token signature and nonce.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != a.Nonce {
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
		Provider: a.Name(),
		Role:     browser.External,
	}

	if len(claims.Roles) >= 1 {
		u.Role = browser.NewRole(claims.Roles[0])
	}

	return u, nil
}
