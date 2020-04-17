// Copyright 2020 Eurac Research. All rights reserved.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gitlab.inf.unibz.it/lter/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/access"
	"gitlab.inf.unibz.it/lter/browser/internal/http"
	"gitlab.inf.unibz.it/lter/browser/internal/influx"
	"gitlab.inf.unibz.it/lter/browser/internal/oauth2"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/peterbourgon/ff"
)

const defaultAddr = "localhost:8888" // default webserver address

func main() {
	log.SetPrefix("browser: ")

	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	var (
		httpAddr       = fs.String("http", defaultAddr, "HTTP service address.")
		influxAddr     = fs.String("influx-addr", "http://127.0.0.1:8086", "Influx (http:https)://host:port")
		influxUser     = fs.String("influx-username", "", "Influx username")
		influxPass     = fs.String("influx-password", "", "Influx password")
		influxDatabase = fs.String("influx-database", "", "Influx database name")
		snipeitAddr    = fs.String("snipeit-addr", "", "SnipeIT API URL")
		snipeitToken   = fs.String("snipeit-token", "", "SnipeIT API Token")
		oauthClientID  = fs.String("oauth-clientid", "", "")
		oauthSecret    = fs.String("oauth-secret", "", "")
		oauthRedirect  = fs.String("oauth-redirect", "", "")
		oauthState     = fs.String("oauth-state", "", "Random string for OAuth2 state code.")
		jwtKey         = fs.String("jwt-key", "", "Secret key used to create a JWT. Don't share it.")
		jwtAppNonce    = fs.String("jwt-app-nonce", "", "Random string for JWT verification.")
		xsrfKey        = fs.String("xsrf-key", "d71404b42640716b0050ad187489c128ec3d611179cf14a29ddd6ea0d536a2c1", "Random string used for generating XSRF token.")
		accessFile     = fs.String("access-file", "access.json", "Access file.")
		analyticsCode  = fs.String("analytics-code", "", "Google Analytics Code")
		_              = fs.String("config", "", "Config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("BROWSER"),
	)

	required("influx-addr", *influxAddr)
	required("influx-database", *influxDatabase)
	required("snipeit-addr", *snipeitAddr)
	required("snipeit-token", *snipeitToken)
	required("jwtKey", *jwtKey)

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
	handler := http.NewHandler(
		http.WithDatabase(acl),
		http.WithMetadata(cache),
		http.WithKey(*xsrfKey),
		http.WithAnalyticsCode(*analyticsCode),
	)

	// Wrap Azure Oauth2 Middleware Authentication around.
	az, err := oauth2.NewAzureOAuth2(
		handler,
		&oauth2.Cookie{
			Secret: *jwtKey,
		},
		&oauth2.AzureOptions{
			ClientID:    *oauthClientID,
			Secret:      *oauthSecret,
			RedirectURL: *oauthRedirect,
			State:       *oauthState,
			Nonce:       *jwtAppNonce,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting server on %s\n", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, az))
}

func required(name, value string) {
	if value == "" {
		fmt.Fprintf(os.Stderr, "flag needs an argument: -%s\n\n", name)
		os.Exit(2)
	}
}
