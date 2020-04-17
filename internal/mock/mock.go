// Copyright 2020 Eurac Research. All rights reserved.

// Package mock provides mock implementations.
package mock

import (
	"errors"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
)

// InfluxClient represents a mock implementation of client.Client.
type InfluxClient struct {
	QueryFn func(q client.Query) (*client.Response, error)
}

func (c *InfluxClient) Ping(timeout time.Duration) (time.Duration, string, error) {
	return (1 * time.Second), "Pong", nil
}

func (c *InfluxClient) QueryAsChunk(q client.Query) (*client.ChunkedResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *InfluxClient) Write(bp client.BatchPoints) error { return errors.New("not implemented") }
func (c *InfluxClient) Close() error                      { return errors.New("not implemented") }

func (c *InfluxClient) Query(q client.Query) (*client.Response, error) {
	if q.Database == "" {
		return nil, errors.New("empty database")
	}

	if q.Command == "" {
		return nil, errors.New("empty query")
	}

	return c.QueryFn(q)
}
