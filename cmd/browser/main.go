// Copyright 2019 Eurac Research. All rights reserved.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"gitlab.inf.unibz.it/lter/browser/internal/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/influx"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"

	client "github.com/influxdata/influxdb1-client/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"github.com/peterbourgon/ff"
)

const defaultAddr = "localhost:8888" // default webserver address

func main() {
	log.SetPrefix("browser: ")

	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	var (
		httpAddr       = fs.String("http", defaultAddr, "HTTP service address")
		influxAddr     = fs.String("influx-addr", "http://127.0.0.1:8086", "Influx (http:https)://host:port")
		influxUser     = fs.String("influx-username", "", "Influx username")
		influxPass     = fs.String("influx-password", "", "Influx password")
		influxDatabase = fs.String("influx-database", "lter_dqc", "Influx database name")
		snipeitAddr    = fs.String("snipeit-addr", "", "SnipeIT API URL")
		snipeitToken   = fs.String("snipeit-token", "", "SnipeIT API Token")
		oauthClientID  = fs.String("oauth-clientid", "", "")
		oauthSecret    = fs.String("oauth-secret", "", "")
		oauthRedirect  = fs.String("oauth-redirect", "", "")
		_              = fs.String("config", "", "Config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("BROWSER"),
	)

	// InfluxDB client
	ic, err := influx.New(client.HTTPConfig{
		Addr:     *influxAddr,
		Username: *influxUser,
		Password: *influxPass,
	}, *influxDatabase)
	if err != nil {
		log.Fatalf("influx: could not create client: %v\n", err)
	}

	// SnipeIT API client
	sc, err := snipeit.NewClient(*snipeitAddr, *snipeitToken)
	if err != nil {
		log.Fatalf("snipeIT: could not create client: %v\n", err)
	}

	// ScientifcNET OAuth2
	oauthConfig := &oauth2.Config{
		ClientID:     *oauthClientID,
		ClientSecret: *oauthSecret,
		Scopes:       []string{"https://graph.microsoft.com/.default"},
		Endpoint:     microsoft.AzureADEndpoint("scientificnet.onmicrosoft.com"),
		RedirectURL:  *oauthRedirect,
	}

	ds := browser.NewDatastore(sc, ic)
	s := auth.Handler(browser.NewServer(ds), oauthConfig)

	log.Printf("Starting server on %s\n", *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, s))
}
