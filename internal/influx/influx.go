// Copyright 2019 Eurac Research. All rights reserved.
package influx

import (
	"errors"
	"fmt"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
)

type Client struct {
	influx   client.Client
	cfg      client.HTTPConfig
	database string
}

func New(cfg client.HTTPConfig, database string) (*Client, error) {
	c, err := client.NewHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	_, _, err = c.Ping(10 * time.Second)
	if err != nil {
		return nil, err
	}

	return &Client{c, cfg, database}, nil
}

func (c *Client) Results(query string) ([]client.Result, error) {
	q := client.NewQuery(query, c.database, "")

	resp, err := c.influx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	if resp.Error() != nil {
		return nil, fmt.Errorf("%v", resp.Error())
	}

	return resp.Results, nil
}

func (c *Client) Result(query string) (client.Result, error) {
	resp, err := c.Results(query)
	if err != nil {
		return client.Result{}, err
	}

	if len(resp) > 1 {
		return client.Result{}, errors.New("multiple queries per request not supported")
	}

	if len(resp) == 0 {
		return client.Result{}, errors.New("no results found")
	}

	return resp[0], nil
}
