package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type contextKey string

type Role string

const (
	Public     Role = "Public"
	FullAccess      = "FullAccess"

	// JWTClaimsContextKey holds the key used to store the JWT Claims in the
	// context.
	JWTClaimsContextKey contextKey = "BrowserLTER"

	jwtCookieName = "browser-session"
)

var (
	// ErrTokenInvalid denotes a token was not able to be validated.
	ErrTokenInvalid = errors.New("JWT Token was invalid")
)

func ParseRole(s string) Role {
	switch s {
	default:
		return Public
	case "FullAccess":
		return FullAccess
	}
}

// JWTClaims denotes a custom JWT claim.
type JWTClaims struct {
	Role string `json:"grp"`
	jwt.StandardClaims
}

// NewJWT creates and returns a new JWT token from the give key and stores
// it in an cookie.
func NewJWT(key []byte, role Role, w http.ResponseWriter) (string, error) {
	exp := time.Now().Add(time.Hour * 48)

	// Create the Claims
	claims := JWTClaims{
		string(role),
		jwt.StandardClaims{
			ExpiresAt: exp.Unix(),
			Issuer:    "BrowserLTER",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	cookie := &http.Cookie{
		Name:    jwtCookieName,
		Value:   tokenString,
		Path:    "/",
		Expires: exp,
	}

	http.SetCookie(w, cookie)

	return tokenString, nil
}

func IsAuthenticated(r *http.Request) bool {
	_, err := r.Cookie(jwtCookieName)
	if err != nil {
		return false
	}
	return true
}

// RoleFromJWT reads an JWT token from a cookie, checks if it is valid
// and returns the claims group value.
func RoleFromJWT(key []byte, r *http.Request) (Role, error) {
	c, err := r.Cookie(jwtCookieName)
	if err != nil {
		return Public, err
	}

	token, err := jwt.ParseWithClaims(c.Value, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return Public, err
	}

	// Vaildate it.
	claims, ok := token.Claims.(*JWTClaims)
	if !ok && !token.Valid {
		return Public, ErrTokenInvalid
	}

	return ParseRole(claims.Role), nil
}
