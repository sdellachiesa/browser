package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

func Handler(next http.Handler, cfg *oauth2.Config) http.Handler {
	state, err := generateRandomToken(32)
	if err != nil {
		log.Println(err)
		state = "randomwords"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/sns":
			http.Redirect(w, r, cfg.AuthCodeURL(state), http.StatusMovedPermanently)
			return
		case "/login/sns/callback":
			if r.FormValue("state") != state {
				http.Error(w, fmt.Sprintf("invalid state token, got %q, want %q.", r.FormValue("state"), state), http.StatusInternalServerError)
				return
			}

			ctx := context.Background()
			token, err := cfg.Exchange(ctx, r.URL.Query().Get("code"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Println(token.AccessToken)
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
		}

		next.ServeHTTP(w, r)
	})
}

func generateRandomToken(s int) (string, error) {
	b := make([]byte, s)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
