// Copyright 2020 Eurac Research. All rights reserved.

// Package browser is the root package for the browser web
// application and contains all domain types.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// DefaultCollectionInterval is the default interval with which LTER stations
// aggregate measured points.
const DefaultCollectionInterval = 15 * time.Minute

var (
	ErrAuthentication = errors.New("user not authenticated")
	ErrDataNotFound   = errors.New("no data points")
	ErrInternal       = errors.New("internal error")
	ErrInvalidToken   = errors.New("invalid token")

	// Location denotes the time location of the LTER stations, which is
	// UTC+1.
	Location = time.FixedZone("+0100", 60*60)
)

// Station represents a meteorological station of the LTER
// project with it's associated metadata and a list of
// measurements.
type Station struct {
	ID           string
	Name         string
	Landuse      string
	Elevation    int64
	Latitude     float64
	Longitude    float64
	Image        string
	Measurements []string
}

// Stations represents a group of meteorological stations.
type Stations []*Station

// String converts stations to a JSON string.
func (s Stations) String() string {
	j, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}

	return string(j)
}

// Get returns the station by given id. If no station is
// found it will return nil and false for indicating that no
// station was found.
func (s Stations) Get(id string) (*Station, bool) {
	for _, station := range s {
		if id == station.ID {
			return station, true
		}
	}
	return nil, false
}

// Landuse returns a sorted list of the landuse of all stations,
// removing duplicates.
func (s Stations) Landuse() []string {
	var l []string

	for _, station := range s {
		l = unique(l, station.Landuse)
	}

	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	return l
}

// Measurements returns a sorted list of all measurements of all
// stations, removing duplicates.
func (s Stations) Measurements() []string {
	var v []string

	for _, station := range s {
		for _, f := range station.Measurements {
			v = unique(v, f)
		}
	}

	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })

	return v
}

// unique removes duplicate values of s from the given slice
// and returns a new slice.
func unique(slice []string, s string) []string {
	for _, el := range slice {
		if el == s {
			return slice
		}
	}
	return append(slice, s)
}

// TimeSeries represents a group Measurements.
type TimeSeries []*Measurement

// Measurement represents a single measurements.
type Measurement struct {
	Label       string
	Station     string
	Aggregation string
	Unit        string
	Landuse     string
	Elevation   int64
	Depth       int64
	Latitude    float64
	Longitude   float64
	Points      []*Point
}

// Name returns the label removing the aggregation function from it.
func (m *Measurement) Name() string {
	// Remove depth from the label if the measurment has a depth.
	if m.Depth > 0 {
		return strings.ReplaceAll(m.Label, fmt.Sprintf("_%02d_%s", m.Depth, m.Aggregation), "")
	}
	return strings.ReplaceAll(m.Label, "_"+m.Aggregation, "")
}

// DepthToString will return the depth as string.
func (m *Measurement) DepthToString() string {
	if m.Depth == 0 {
		return ""
	}

	return fmt.Sprint(m.Depth)
}

// Point represents a single measured point.
type Point struct {
	Timestamp time.Time
	Value     float64
}

// Message represents a message exchange between services.
type Message struct {
	Stations     []string
	Measurements []string
	Landuse      []string
	Limit        int64

	Start time.Time
	End   time.Time
}

// Stmt is a query statement composed of the actual query and
// the database it is performed on.
type Stmt struct {
	Query    string
	Database string
}

// Metadata represents a backend for retrieving Metadata.
type Metadata interface {
	// Stations retrieves metadata about all stations.
	Stations(ctx context.Context, m *Message) (Stations, error)
}

// Database represents a backend for retrieving time series data.
type Database interface {
	// Series returns a TimeSeries from the given Message. Points in a
	// TimeSeries should always have a continuous time range as for
	// https://gitlab.inf.unibz.it/lter/browser/issues/10
	Series(ctx context.Context, m *Message) (TimeSeries, error)

	// Query returns a query Stmt for the given Message.
	Query(ctx context.Context, m *Message) *Stmt
}

// Role represents a role a User is part of.
type Role string

const (
	Public      Role = "Public"
	FullAccess  Role = "FullAccess"
	DefaultRole Role = Public
)

// Roles is a list of all supported Roles.
var Roles = []Role{Public, FullAccess}

func (r *Role) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*r = NewRole(s)
	return nil
}

// NewRole returns a new role from the given string. If the string
// cannot be parsed to a role the default group will be returned.
func NewRole(s string) Role {
	switch s {
	default:
		return DefaultRole

	case "FullAccess":
		return FullAccess
	}
}

// User represents a specific user.
type User struct {
	Username string
	Name     string
	Role     Role
}

// contextKey is a custom type to be used as key type for context.Context
// values.
type contextKey string

// BrowserContextKey holds the key used to store in the current context.
const BrowserContextKey contextKey = "BrowserLTER"

// UserFromContext reads user information from the given context. If
// the context has no user information a default user will be
// returned.
func UserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(BrowserContextKey).(*User)
	if !ok {
		return &User{
			Username: "",
			Name:     "",
			Role:     DefaultRole,
		}
	}
	return user
}

// Authenticator represents a service for authenticating users.
type Authenticator interface {
	// Validate returns an authenticated User if a valid user
	// session is found.
	Validate(context.Context, *http.Request) (*User, error)

	// Authorize will create a new user session for authenticated users.
	Authorize(ctx context.Context, w http.ResponseWriter, u *User) error

	// Expire will logout the authenticated User.
	Expire(http.ResponseWriter)
}
