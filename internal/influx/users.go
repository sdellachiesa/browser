// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package influx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/euracresearch/browser"
	client "github.com/influxdata/influxdb1-client/v2"
)

// Guarantee we implement browser.UserService.
var _ browser.UserService = &UserService{}

// UserService represents an service for retrieving information of an
// authenticated user stored in InfluxDB.
type UserService struct {
	Client   client.Client
	Database string
	Env      string
}

// user represents an browser.User with additional information.
type user struct {
	*browser.User

	created time.Time
}

// Get returns the given user if found.
func (s *UserService) Get(ctx context.Context, user *browser.User) (*browser.User, error) {
	if user == nil || !user.Valid() {
		return nil, browser.ErrUserNotFound
	}

	u, err := s.get(user)
	if err != nil {
		return nil, err
	}

	return &browser.User{
		Name:     u.Name,
		Email:    u.Email,
		Picture:  u.Picture,
		Provider: u.Provider,
		License:  u.License,
		Role:     u.Role,
	}, nil
}

func (s *UserService) get(u *browser.User) (*user, error) {
	q := fmt.Sprintf("SELECT updated FROM %s WHERE email='%s' and provider='%s' GROUP BY provider,fullname,email,picture,license,role",
		s.Env,
		u.Email,
		u.Provider,
	)

	resp, err := s.Client.Query(client.NewQuery(q, s.Database, ""))
	if err != nil {
		return nil, err
	}
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	switch {
	case len(resp.Results) != 1:
		return nil, browser.ErrUserNotFound
	case len(resp.Results[0].Series) != 1:
		return nil, browser.ErrUserNotFound
	}

	tags := resp.Results[0].Series[0].Tags
	lic, err := strconv.ParseBool(tags["license"])
	if err != nil {
		lic = false
	}

	var created time.Time
	for _, v := range resp.Results[0].Series[0].Values {
		t, err := time.Parse(time.RFC3339, v[0].(string))
		if err != nil {
			return nil, err
		}
		created = t
	}

	return &user{
		&browser.User{
			Name:     tags["fullname"],
			Email:    tags["email"],
			Picture:  tags["picture"],
			Provider: tags["provider"],
			License:  lic,
			Role:     browser.NewRole(tags["role"]),
		},

		created,
	}, nil
}

// Create adds a new user to the database.
func (s *UserService) Create(ctx context.Context, user *browser.User) error {
	if user == nil || !user.Valid() {
		return browser.ErrUserNotValid
	}
	_, err := s.Get(ctx, user)
	if err == nil {
		return browser.ErrUserAlreadyExists
	}

	return s.create(user, time.Now())
}

func (s *UserService) create(user *browser.User, ts time.Time) error {
	p, err := client.NewPoint(
		s.Env,
		map[string]string{
			"provider": user.Provider,
			"fullname": user.Name,
			"email":    user.Email,
			"picture":  user.Picture,
			"license":  strconv.FormatBool(user.License),
			"role":     string(user.Role),
		},
		map[string]interface{}{
			"updated": time.Now().Unix(),
		},
		ts,
	)
	if err != nil {
		return err
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{Database: s.Database})
	if err != nil {
		return err
	}
	bp.AddPoint(p)

	return s.Client.Write(bp)
}

// Update will update the user information stored in the database. In Influx we
// cannot update single entries so we first need to retrieve the current stored
// user, delete it and re-create it with the given user.
func (s *UserService) Update(ctx context.Context, user *browser.User) error {
	if user == nil || !user.Valid() {
		return browser.ErrUserNotValid
	}

	dbuser, err := s.get(user)
	if err != nil {
		return err
	}

	if err := s.delete(dbuser); err != nil {
		return err
	}

	return s.create(user, dbuser.created)
}

// Delete will delete the given user from the database.
func (s *UserService) Delete(ctx context.Context, user *browser.User) error {
	if user == nil || !user.Valid() {
		return browser.ErrUserNotValid
	}

	dbuser, err := s.get(user)
	if err != nil {
		return err
	}

	return s.delete(dbuser)
}

func (s *UserService) delete(dbuser *user) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE email='%s' AND provider='%s' AND time=%d",
		s.Env,
		dbuser.Email,
		dbuser.Provider,
		dbuser.created.UnixNano(),
	)

	resp, err := s.Client.Query(client.NewQuery(q, s.Database, ""))
	if err != nil {
		return err
	}

	return resp.Error()
}
