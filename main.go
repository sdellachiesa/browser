package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peterbourgon/ff"

	client "github.com/influxdata/influxdb1-client/v2"
)

// TODO: check influx db error response
// TODO: fmt.Sprintf %s for queries composing is not safe (SQL injection)
// TODO: add key to requests for preventing request forgery
// TODO: check on Cross-origin resource sharing (CORS)

const defaultAddr = "localhost:8888" // default webserver address

func main() {
	log.SetPrefix("browser: ")

	fs := flag.NewFlagSet("browser", flag.ExitOnError)
	var (
		httpAddr   = fs.String("http", defaultAddr, "HTTP service address")
		influxAddr = fs.String("influx-addr", "http://127.0.0.1:8086", "Influx protocol:\\host:port")
		influxUser = fs.String("influx-username", "", "Influx username")
		influxPass = fs.String("influx-password", "", "Influx password")
		dbName     = fs.String("influx-database", "lter", "Influx database name")
		//oauthClientID = fs.String("oauth-clientid", "", "")
		//oauthSecret   = fs.String("oauth-secret", "", "")
		//oauthRedirect = fs.String("oauth-redirect", "", "")
		_ = fs.String("config", "", "Config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
		ff.WithEnvVarPrefix("BROWSER"),
	)

	s, err := newServer(func(s *server) error {
		c, err := client.NewHTTPClient(client.HTTPConfig{
			Addr:     *influxAddr,
			Username: *influxUser,
			Password: *influxPass,
		})
		if err != nil {
			return fmt.Errorf("could not create influx client: %v", err)
		}
		defer c.Close()

		_, _, err = c.Ping(10 * time.Second)
		if err != nil {
			return fmt.Errorf("error connecting to influx: %v", err)
		}
		s.db = c
		s.database = *dbName

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(*httpAddr, s))
}

func generateKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
