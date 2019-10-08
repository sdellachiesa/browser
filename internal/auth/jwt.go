package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

type contextKey string

const (
	// JWTClaimsContextKey holds the key used to store the JWT Claims in the
	// context.
	JWTClaimsContextKey contextKey = "BrowserLTER"

	jwtCookieName = "browser-session"
)

var (
	// ErrTokenInvalid denotes a token was not able to be validated.
	ErrTokenInvalid = errors.New("JWT Token was invalid")
)

// JWTClaims denotes a custom JWT claim.
type JWTClaims struct {
	Group string `json:"grp"`
	jwt.StandardClaims
}

// NewJWT creates and returns a new JWT token from the give key and stores
// it in an cookie.
func NewJWT(key []byte, group string, w http.ResponseWriter) (string, error) {
	// Create the Claims
	claims := JWTClaims{
		group,
		jwt.StandardClaims{
			ExpiresAt: 15000,
			Issuer:    "BrowserLTER",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	// TODO: set expiry date?
	cookie := &http.Cookie{
		Name:     jwtCookieName,
		Value:    tokenString,
		HttpOnly: true,
		Path:     "/",
	}

	http.SetCookie(w, cookie)

	return tokenString, nil
}

// readJWTCookie will try to read the JWT cookie and if
// successfully read return the signed token. On any
// error it will return a newly created signed token,
// with default claims.
func readJWTCookie(key []byte, w http.ResponseWriter, r *http.Request) (string, error) {
	c, err := r.Cookie(jwtCookieName)
	if err != nil {
		log.Println(err)
		return NewJWT(key, "Public", w)

	}
	return c.Value, nil
}

// GetJWTClaims read an JWT token from a cookie, decrypt it with
// the given key and return it's claims.
func GetJWTClaims(key []byte, w http.ResponseWriter, r *http.Request) (*JWTClaims, error) {
	tokenString, err := readJWTCookie(key, w, r)
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	// Vaildate it.
	claims, ok := token.Claims.(*JWTClaims)
	if !ok && !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
