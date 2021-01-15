// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package mock provides mock implementations of various interfaces used only
// for testing.
package mock

import (
	"context"
	"errors"
	"time"

	"github.com/euracresearch/browser"
	client "github.com/influxdata/influxdb1-client/v2"
)

// InfluxClient represents a mock implementation of client.Client.
type InfluxClient struct {
	QueryFn func(q client.Query) (*client.Response, error)
	WriteFn func(bp client.BatchPoints) error
}

func (c *InfluxClient) Ping(timeout time.Duration) (time.Duration, string, error) {
	return (1 * time.Second), "Pong", nil
}

func (c *InfluxClient) QueryAsChunk(q client.Query) (*client.ChunkedResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *InfluxClient) Write(bp client.BatchPoints) error {
	return c.WriteFn(bp)
}

func (c *InfluxClient) Close() error { return errors.New("not implemented") }

func (c *InfluxClient) Query(q client.Query) (*client.Response, error) {
	if q.Database == "" {
		return nil, errors.New("empty database")
	}

	if q.Command == "" {
		return nil, errors.New("empty query")
	}

	return c.QueryFn(q)
}

// Database represents a mock implementation of browser.Database.
type Database struct {
	QueryFn  func(ctx context.Context, m *browser.SeriesFilter) *browser.Stmt
	SeriesFn func() (browser.TimeSeries, error)
}

func (db *Database) Series(ctx context.Context, m *browser.SeriesFilter) (browser.TimeSeries, error) {
	return db.SeriesFn()
}

func (db *Database) Query(ctx context.Context, m *browser.SeriesFilter) *browser.Stmt {
	return db.QueryFn(ctx, m)
}
