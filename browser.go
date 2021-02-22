// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package browser is the root package for the browser web application and
// contains all domain types.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultCollectionInterval is the default interval with which LTER stations
// aggregate measured points.
const DefaultCollectionInterval = 15 * time.Minute

var (
	ErrAuthentication    = errors.New("user not authenticated")
	ErrDataNotFound      = errors.New("no data points")
	ErrInternal          = errors.New("internal error")
	ErrInvalidToken      = errors.New("invalid token")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserNotValid      = errors.New("user is not valid")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrGroupsNotFound    = errors.New("no groups found")

	// Location denotes the time location of the LTER stations, which is UTC+1.
	Location = time.FixedZone("+0100", 60*60)
)

// Measurement represents a single measurements with metadata and its points.
type Measurement struct {
	Label       string
	Aggregation string
	Unit        string
	Depth       int64
	Station     *Station
	Points      []*Point
}

// Point represents a single measured point.
type Point struct {
	Timestamp time.Time
	Value     float64
}

// TimeSeries represents a group Measurements.
type TimeSeries []*Measurement

// Database represents a backend for retrieving time series data.
type Database interface {
	// Series returns a TimeSeries filtered with the given SeriesFilter. Points
	// in a TimeSeries should always have a continuous time range as for
	// https://gitlab.inf.unibz.it/lter/browser/issues/10
	Series(context.Context, *SeriesFilter) (TimeSeries, error)

	// GroupsByStation will return a slice of groupped measurements stored in
	// the Database for the given station.
	GroupsByStation(context.Context, int64) ([]Group, error)

	// Maintenance will return a list of measurement names which correspond to
	// maintenance observations.
	Maintenance(context.Context) ([]string, error)

	// Query returns a query Stmt for the given SeriesFilter.
	Query(context.Context, *SeriesFilter) *Stmt
}

// Stmt is a query statement composed of the actual query and the database it is
// performed on.
type Stmt struct {
	Query    string
	Database string
}

// SeriesFilter represents a filter for filtering TimeSeries.
type SeriesFilter struct {
	Groups   []Group
	Stations []string
	Landuse  []string
	Start    time.Time
	End      time.Time

	// WithSTD determines if the Series should contain standard deviations.
	WithSTD bool

	// Maintenance is a list of raw label names corresponding to measurements
	// used for maintenance technicians.
	Maintenance []string
}

// ParseSeriesFilterFromRequest parses form values from the given http.Request
// and returns a a valid SeriesFilter or an error. It performs basic validation
// for the given dates.
func ParseSeriesFilterFromRequest(r *http.Request) (*SeriesFilter, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	start, err := time.ParseInLocation("2006-01-02", r.FormValue("startDate"), Location)
	if err != nil {
		return nil, fmt.Errorf("could not parse start date %v", err)
	}

	end, err := time.ParseInLocation("2006-01-02", r.FormValue("endDate"), Location)
	if err != nil {
		return nil, fmt.Errorf("could not parse end date %v", err)
	}

	if end.After(time.Now()) {
		return nil, errors.New("error: end date is in the future")
	}

	if r.Form["measurements"] == nil {
		return nil, errors.New("at least one measurement must be given")
	}

	if r.Form["stations"] == nil {
		return nil, errors.New("at least one station must be given")
	}

	groups, maint := parseGroupsAndMaintenance(r.Form["measurements"])

	return &SeriesFilter{
		Groups:      groups,
		Stations:    r.Form["stations"],
		Landuse:     r.Form["landuse"],
		Start:       start,
		End:         end,
		Maintenance: maint,
	}, nil
}

// parseGroupsAndMaintenance will parse each string in the given string slice
// into a group and return a unique slice of Groups. If parsing to a group fails
// it will assume the string is a maintenance parameter and add it to the
// returning string slice.
func parseGroupsAndMaintenance(str []string) ([]Group, []string) {
	var g []Group
	var m []string

	for _, s := range str {
		i, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			m = append(m, strings.ToLower(s))
			continue
		}

		g = AppendGroupIfMissing(g, Group(i))
	}

	return g, m
}

// Role represents a role a User is part of.
type Role string

const (
	Public      Role = "Public"
	FullAccess  Role = "FullAccess"
	External    Role = "External"
	DefaultRole Role = Public
)

// Roles is a list of all supported Roles.
var Roles = []Role{Public, External, FullAccess}

func (r *Role) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*r = NewRole(s)
	return nil
}

// NewRole returns a new role from the given string. If the string cannot be
// parsed to a role the default role will be returned.
func NewRole(s string) Role {
	switch s {
	default:
		return DefaultRole

	case "External":
		return External

	case "FullAccess":
		return FullAccess
	}
}

// User represents an authenticated user.
type User struct {
	Name     string
	Email    string
	Picture  string
	Provider string
	License  bool
	Role     Role
}

// Valid determines if a user is valid. A valid user must have a username, name
// and email.
func (u *User) Valid() bool {
	if u.Name != "" && u.Email != "" && u.Provider != "" {
		return true
	}
	return false
}

// UserService is the storage and retrieval of authentication information.
type UserService interface {
	// Get retrieves a user if it exists
	Get(context.Context, *User) (*User, error)
	// Create a new User in the UsersStore
	Create(context.Context, *User) error
	// Delete the user from the UsersStore
	Delete(context.Context, *User) error
	// Update updates the given user
	Update(context.Context, *User) error
}

// userContextKey is a custom type to be used as key type for context.Context
// values.
type userContextKey string

// UserContextKey is the context key for retrieving the user off of context.
const UserContextKey userContextKey = "BrowserLTER"

// UserFromContext reads user information from the given context. If the context
// has no user information a default user will be returned.
func UserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return &User{
			Role:    DefaultRole,
			License: false,
		}
	}
	return user
}
