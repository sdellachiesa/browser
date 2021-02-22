// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/euracresearch/browser"
	"github.com/euracresearch/browser/internal/access"
	"github.com/euracresearch/browser/internal/http"
	"github.com/euracresearch/browser/internal/influx"
	"github.com/euracresearch/browser/internal/middleware"
	"github.com/euracresearch/browser/internal/oauth2"
	"github.com/euracresearch/browser/internal/snipeit"

	"github.com/gorilla/securecookie"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/peterbourgon/ff"
)

const defaultAddr = "localhost:8888" // default webserver address

func main() {
	log.SetPrefix("browser: ")

	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	var (
		listenAddr        = fs.String("listen", defaultAddr, "Server listen address.")
		https             = fs.Bool("https", false, "Serve HTTPS.")
		domain            = fs.String("domain", "", "Domain used for getting LetsEncrypt certificate.")
		influxAddr        = fs.String("influx.addr", "http://127.0.0.1:8086", "Influx (http:https)://host:port")
		influxUser        = fs.String("influx.username", "", "Influx username")
		influxPass        = fs.String("influx.password", "", "Influx password")
		influxDatabase    = fs.String("influx.database", "", "Influx database name")
		usersDatabase     = fs.String("users.database", "", "Database name for storing user information.")
		usersEnvironment  = fs.String("users.env", "testing", "The environment the app is running.")
		snipeitAddr       = fs.String("snipeit.addr", "", "SnipeIT API URL")
		snipeitToken      = fs.String("snipeit.token", "", "SnipeIT API Token")
		jwtKey            = fs.String("jwt.key", "", "Secret key used to create a JWT. Don't share it.")
		xsrfKey           = fs.String("xsrf.key", "d71404b42640716b0050ad187489c128ec3d611179cf14a29ddd6ea0d536a2c1", "Random string used for generating XSRF token.")
		accessFile        = fs.String("access.file", "/etc/browser/access.json", "Access file.")
		analyticsCode     = fs.String("analytics.code", "", "Google Analytics Code")
		cookieHashKey     = fs.String("cookie.hash", "3998130314e70d9037e05bf872881156da20e07f344f6d9ae58f92e4be85a07dbdb8949c2eee7e0498247176df3d7785200e586c1b52b7f87210119297f77552", "Hash key used for securing the HTTP cookie. Should be at least 32 bytes long.")
		cookieBlockKey    = fs.String("cookie.block", "e48f59d35c3871586f68d788bcff6c45", "Block keys should be 16 bytes (AES-128) or 32 bytes (AES-256) long. Shorter keys may weaken the encryption used.")
		oauthState        = fs.String("oauth2.state", "", "Random string used for OAuth2 state code.")
		oauthNonce        = fs.String("oauth2.nonce", "", "Random string for ID token verification.")
		microsoftClientID = fs.String("microsoft.clientid", "", "Microsoft OAuth2 client ID.")
		microsoftSecret   = fs.String("microsoft.secret", "", "Microsoft OAuth2 secret.")
		microsoftRedirect = fs.String("microsoft.redirect", "", "Microsoft OAuth2 redirect URL.")
		githubClientID    = fs.String("github.clientid", "", "Github OAuth2 client ID.")
		githubSecret      = fs.String("github.secret", "", "Github OAuth2 secret.")
		googleClientID    = fs.String("google.clientid", "", "Google OAuth2 client ID.")
		googleSecret      = fs.String("google.secret", "", "Google OAuth2 secret.")
		googleRedirect    = fs.String("google.redirect", "", "Google OAuth2 redirect URL.")
		_                 = fs.String("config", "", "Config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("BROWSER"),
	)

	required("influx.addr", *influxAddr)
	required("influx.database", *influxDatabase)
	required("users.database", *usersDatabase)
	required("snipeit.addr", *snipeitAddr)
	required("snipeit.token", *snipeitToken)
	required("jwt.key", *jwtKey)

	// Initialize influx v1 client.
	ic, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     *influxAddr,
		Username: *influxUser,
		Password: *influxPass,
	})
	if err != nil {
		log.Fatalf("influx: could not create client: %v\n", err)
	}
	defer ic.Close()

	_, _, err = ic.Ping(10 * time.Second)
	if err != nil {
		log.Fatalf("influx: could not contact Influx DB: %v\n", err)
	}

	// Initialize services.
	db := influx.NewDB(ic, *influxDatabase)
	metadata, err := snipeit.NewSnipeITService(*snipeitAddr, *snipeitToken, ic, *influxDatabase)
	if err != nil {
		log.Fatal(err)
	}

	// Decorating the Database and Metadata with an ACL service.
	acl, err := access.New(*accessFile, db, metadata)
	if err != nil {
		log.Fatal(err)
	}

	// Decorating the Metadata service with an in memory cache service.
	cache := browser.NewInMemCache(acl)

	// Initialize HTTP endpoints.
	frontend := http.NewHandler(
		http.WithDatabase(acl),
		http.WithMetadata(cache),
		http.WithAnalyticsCode(*analyticsCode),
	)

	// Initialize authentication handler.
	handler := &oauth2.Handler{
		Next:  frontend,
		State: *oauthState,
		Nonce: *oauthNonce,
		Auth: &oauth2.Cookie{
			Secret: *jwtKey,
			Cookie: securecookie.New([]byte(*cookieHashKey), []byte(*cookieBlockKey)),
		},
		Users: &influx.UserService{
			Client:   ic,
			Database: *usersDatabase,
			Env:      *usersEnvironment,
		},
	}

	// Initialize OAuth2 providers.
	handler.Register(&oauth2.Microsoft{
		Provider:    "microsoft",
		ClientID:    *microsoftClientID,
		Secret:      *microsoftSecret,
		RedirectURL: *microsoftRedirect,
		Nonce:       *oauthNonce,
	})

	handler.Register(&oauth2.Github{
		ClientID: *githubClientID,
		Secret:   *githubSecret,
	})

	handler.Register(&oauth2.Google{
		ClientID:    *googleClientID,
		Secret:      *googleSecret,
		RedirectURL: *googleRedirect,
		Nonce:       *oauthNonce,
	})

	// Add some common middleware.
	mw := middleware.Chain(
		middleware.SecureHeaders(),
		middleware.XSRFProtect(*xsrfKey),
	)

	log.Printf("Starting server on %s\n", *listenAddr)
	if *https && *domain != "" {
		log.Fatal(http.ServeAutoCert(*listenAddr, mw(handler), *domain))
	}

	log.Fatal(http.ListenAndServe(*listenAddr, mw(handler)))
}

func required(name, value string) {
	if value == "" {
		fmt.Fprintf(os.Stderr, "flag needs an argument: -%s\n\n", name)
		os.Exit(2)
	}
}
