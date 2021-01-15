// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/euracresearch/browser"
	"github.com/gorilla/securecookie"
)

const (
	// DefaultCookieName is the name of the stored cookie
	DefaultCookieName = "browser_lter_session"

	// DefaultLifespan is the duration a token and cookie is valid
	DefaultLifespan = 48 * time.Hour

	// DefaultJWTIssure is the default issure of the JWT token
	DefaultJWTIssuer = "BrowserLTER"
)

var (
	// Guarantee we implement Authenticator.
	_ Authenticator = &Cookie{}

	// ErrTokenInvalid denotes that a could not be validated.
	ErrTokenInvalid = errors.New("token is invalid")
)

// Cookie is an Authenticator using HTTP cookies and JWT tokens.
type Cookie struct {
	// Secret used for JWT generation/validation.
	Secret string
	// Cookie used for storing JWT token in a secure manner.
	Cookie *securecookie.SecureCookie
}

func (c *Cookie) Authorize(ctx context.Context, w http.ResponseWriter, u *browser.User) error {
	if u == nil {
		return browser.ErrAuthentication
	}

	token, err := c.newJWT(u)
	if err != nil {
		return err
	}

	encoded, err := c.Cookie.Encode(DefaultCookieName, token)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:    DefaultCookieName,
		Value:   encoded,
		Path:    "/",
		Expires: time.Now().Add(DefaultLifespan),
	})

	return nil
}

func (c *Cookie) Expire(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:    DefaultCookieName,
		Value:   "none",
		Path:    "/",
		Expires: time.Now().Add(-1 * time.Hour),
	}

	http.SetCookie(w, cookie)
}

// Validate validates the JWT token stored in the cookie and return the user
// information. It will not validate the user against the user service.
func (c *Cookie) Validate(ctx context.Context, r *http.Request) (*browser.User, error) {
	cookie, err := r.Cookie(DefaultCookieName)
	if err != nil {
		return nil, err
	}

	var value string
	if err := c.Cookie.Decode(DefaultCookieName, cookie.Value, &value); err != nil {
		return nil, err
	}

	u, err := c.validateJWT(value)
	if err != nil {
		return nil, err
	}

	return u, nil
}

type claims struct {
	User *browser.User
	jwt.StandardClaims
}

// newJWT creates a new signed JWT token with the given user information
// embedded.
func (c *Cookie) newJWT(u *browser.User) (string, error) {
	if u == nil {
		return "", errors.New("no user provided")
	}

	id, err := generateKey()
	if err != nil {
		return "", err
	}

	date := time.Now()
	exp := date.Add(DefaultLifespan)

	cl := claims{
		u,
		jwt.StandardClaims{
			Issuer:    DefaultJWTIssuer,
			IssuedAt:  date.Unix(),
			Id:        id,
			NotBefore: date.Unix(),
			ExpiresAt: exp.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, cl)

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(c.Secret))
}

func (c *Cookie) validateJWT(token string) (*browser.User, error) {
	t, err := jwt.ParseWithClaims(token, &claims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.Secret), nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Vaildate it.
	cl, ok := t.Claims.(*claims)
	if !ok && !t.Valid {
		return nil, ErrTokenInvalid
	}

	// Validates time based claims "exp, iat, nbf".
	if err := cl.Valid(); err != nil {
		return nil, ErrTokenInvalid
	}

	if !cl.VerifyIssuer(DefaultJWTIssuer, true) {
		return nil, ErrTokenInvalid
	}

	return cl.User, nil
}

func generateKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
