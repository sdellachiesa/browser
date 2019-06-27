package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peterbourgon/ff"
	"gitlab.inf.unibz.it/lter/browser/internal/snipeit"
	"gitlab.inf.unibz.it/lter/browser/static"

	client "github.com/influxdata/influxdb1-client/v2"
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
		influxDatabase = fs.String("influx-database", "lter", "Influx database name")
		snipeitAddr    = fs.String("snipeit-addr", "", "SnipeIT API URL")
		snipeitToken   = fs.String("snipeit-token", "", "SnipeIT API Token")
		//oauthClientID  = fs.String("oauth-clientid", "", "")
		//oauthSecret   = fs.String("oauth-secret", "", "")
		//oauthRedirect = fs.String("oauth-redirect", "", "")
		_ = fs.String("config", "", "Config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("BROWSER"),
	)

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
		log.Fatalf("influx: error connecting to influx server: %v\n", err)
	}

	// SnipeIT API client
	sc, err := snipeit.NewClient(*snipeitAddr, *snipeitToken)
	if err != nil {
		log.Fatalf("snipeIT: could not create client: %v\n", err)
	}

	m := http.NewServeMux()
	m.Handle("/", static.Handler())
	m.Handle("/api/", NewAPIHandler(&Backend{
		Influx:   ic,
		SnipeIT:  sc,
		Database: *influxDatabase,
	}))

	srv := &http.Server{
		Addr:    *httpAddr,
		Handler: m,
	}

	log.Printf("Starting server on %s\n", *httpAddr)
	log.Fatal(srv.ListenAndServe())
}
