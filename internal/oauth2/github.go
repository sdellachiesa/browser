// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"errors"
	"fmt"

	"gitlab.inf.unibz.it/lter/browser"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

// Guarantee we implement Provider.
var _ Provider = &Github{}

// Github is an OAuth2 provider for signing in using Github accounts.
type Github struct {
	ClientID string
	Secret   string
}

// Name returns the name of the provider.
func (g *Github) Name() string {
	return "github"
}

// Config is the Github OAuth2 configuration.
func (g *Github) Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.Secret,
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
}

// User returns an browser.User with information from Github.
func (g *Github) User(ctx context.Context, token *oauth2.Token) (*browser.User, error) {
	client := github.NewClient(g.Config().Client(ctx, token))

	u, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	if u.Login == nil {
		return nil, errors.New("Github profile is missing an username")
	}
	if u.Name == nil {
		return nil, errors.New("Github profile is missing a name")
	}

	email, err := getEmail(client)
	if err != nil {
		return nil, err
	}

	return &browser.User{
		Name:     *u.Name,
		Email:    email,
		Picture:  *u.AvatarURL,
		Provider: g.Name(),
		Role:     browser.External,
	}, nil
}

func getEmail(client *github.Client) (string, error) {
	ctx := context.Background()
	emails, resp, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error in fetching emails: %v", resp.Status)
	}

	for _, e := range emails {
		if e != nil && isPrimary(e) && isVerified(e) && e.Email != nil {
			return *e.Email, nil
		}
	}

	return "", errors.New("no email found")
}

func isPrimary(e *github.UserEmail) bool {
	if e == nil || e.Primary == nil {
		return false
	}
	return *e.Primary
}

func isVerified(e *github.UserEmail) bool {
	if e == nil || e.Verified == nil {
		return false
	}
	return *e.Verified
}
