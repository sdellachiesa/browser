// Copyright 2019 Eurac Research. All rights reserved.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"gitlab.inf.unibz.it/lter/browser/internal/auth"
	"gitlab.inf.unibz.it/lter/browser/internal/browser"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"

	"github.com/peterbourgon/ff"

	client "github.com/influxdata/influxdb1-client/v2"
)

const defaultAddr = "localhost:8888" // default webserver address

func main() {
	log.SetPrefix("browser: ")

	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	var (
		httpAddr       = fs.String("http", defaultAddr, "HTTP service address.")
		serveTLS       = fs.Bool("tls", false, "Run Browser application as HTTPS.")
		acmeCacheDir   = fs.String("acme-cache", "letsencrypt", "Direcotry for storing letsencrypt certificates.")
		acmeHostname   = fs.String("acme-hostname", "", "Hostname used for getting a letsencrypt certificate.")
		influxAddr     = fs.String("influx-addr", "http://127.0.0.1:8086", "Influx (http:https)://host:port")
		influxUser     = fs.String("influx-username", "", "Influx username")
		influxPass     = fs.String("influx-password", "", "Influx password")
		influxDatabase = fs.String("influx-database", "", "Influx database name")
		snipeitAddr    = fs.String("snipeit-addr", "", "SnipeIT API URL")
		snipeitToken   = fs.String("snipeit-token", "", "SnipeIT API Token")
		oauthClientID  = fs.String("oauth-clientid", "", "")
		oauthSecret    = fs.String("oauth-secret", "", "")
		oauthRedirect  = fs.String("oauth-redirect", "", "")
		jwtKey         = fs.String("jwt-key", "", "Secret key used to create a JWT. Don't share it.")
		accessFile     = fs.String("access-file", "access.json", "Access file.")
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

	// InfluxDB client
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

	// SnipeIT API client
	sc, err := snipeit.NewClient(*snipeitAddr, *snipeitToken)
	if err != nil {
		log.Fatalf("snipeIT: could not create client: %v\n", err)
	}

	// ScientifcNET OAuth2
	oauthConfig := &oauth2.Config{
		ClientID:     *oauthClientID,
		ClientSecret: *oauthSecret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     microsoft.AzureADEndpoint("scientificnet.onmicrosoft.com"),
		RedirectURL:  *oauthRedirect,
	}

	ds := browser.NewDatastore(sc, ic, *influxDatabase)
	a := browser.ParseAccessFile(*accessFile)

	b, err := browser.NewServer(
		browser.WithBackend(ds),
		browser.WithDecoder(a),
		browser.WithInfluxDB(*influxDatabase),
	)
	if err != nil {
		log.Fatalf("Error creating server: %v\n", err)
	}

	handler := auth.Azure(b, oauthConfig, []byte(*jwtKey))

	log.Printf("Starting server on %s\n", *httpAddr)
	if !*serveTLS {
		log.Fatal(http.ListenAndServe(*httpAddr, handler))
	}

	required("acme-hostname", *acmeHostname)
	m := &autocert.Manager{
		Cache:      autocert.DirCache(*acmeCacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*acmeHostname),
	}
	srv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig:    m.TLSConfig(),
		Handler:      handler,
	}
	// Redirect HTTP traffic to HTTPS
	go func() {
		host, _, err := net.SplitHostPort(*httpAddr)
		if err != nil || host == "" {
			host = "0.0.0.0"
		}
		log.Println("Redirecting traffic from HTTP to HTTPS.")
		log.Fatal(http.ListenAndServe(host+":80", redirectHandler()))
	}()

	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func required(name, value string) {
	if value == "" {
		fmt.Fprintf(os.Stderr, "flag needs an argument: -%s\n\n", name)
		os.Exit(2)
	}
}

func redirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		url := "https://" + r.Host + r.URL.String()
		http.Redirect(w, r, url, http.StatusMovedPermanently)
		return
	})
}
