// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ui contains the domain concept definitions needed to support
// Mainflux ui adapter service functionality.
package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

const (
	templateDir = "ui/web/template"
)

var (
	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")
)

// Service specifies coap service API.
type Service interface {
	Index(ctx context.Context, token string) ([]byte, error)
	CreateThings(ctx context.Context, token string, things ...sdk.Thing) ([]byte, error)
	ViewThing(ctx context.Context, token, id string) ([]byte, error)
	UpdateThing(ctx context.Context, token, id string, thing sdk.Thing) ([]byte, error)
	ListThings(ctx context.Context, token string) ([]byte, error)
	RemoveThing(ctx context.Context, token, id string) ([]byte, error)
	CreateChannels(ctx context.Context, token string, channels ...sdk.Channel) ([]byte, error)
	ViewChannel(ctx context.Context, token, id string) ([]byte, error)
	UpdateChannel(ctx context.Context, token, id string, channel sdk.Channel) ([]byte, error)
	ListChannels(ctx context.Context, token string) ([]byte, error)
	RemoveChannel(ctx context.Context, token, id string) ([]byte, error)
	CreateGroups(ctx context.Context, token string, groups ...sdk.Group) ([]byte, error)
	ViewGroup(ctx context.Context, token, id string) ([]byte, error)
	UpdateGroup(ctx context.Context, token, id string, group sdk.Group) ([]byte, error)
	ListGroups(ctx context.Context, token string) ([]byte, error)
	RemoveGroup(ctx context.Context, token, id string) ([]byte, error)
}

var _ Service = (*uiService)(nil)

type uiService struct {
	things mainflux.ThingsServiceClient
	sdk    sdk.SDK
}

// New instantiates the HTTP adapter implementation.
func New(things mainflux.ThingsServiceClient, sdk sdk.SDK) Service {
	return &uiService{
		things: things,
		sdk:    sdk,
	}
}

func (gs *uiService) Index(ctx context.Context, token string) ([]byte, error) {
	tpl := template.New("index")
	tpl = tpl.Funcs(template.FuncMap{
		"toJSON": func(data map[string]interface{}) string {
			ret, _ := json.Marshal(data)
			return string(ret)
		},
	})
	var err error

	tpl, err = tpl.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
	}{
		"dashboard",
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "index", data); err != nil {
		println(err.Error())
	}

	return btpl.Bytes(), nil
}

func (gs *uiService) CreateThings(ctx context.Context, token string, things ...sdk.Thing) ([]byte, error) {

	for i := range things {
		_, err := gs.sdk.CreateThing(things[i], "123")
		if err != nil {
			return []byte{}, err
		}
	}

	return gs.ListThings(ctx, "123")
}

func (gs *uiService) ListThings(ctx context.Context, token string) ([]byte, error) {
	tpl := template.New("things")
	tpl = tpl.Funcs(template.FuncMap{
		"toJSON": func(data map[string]interface{}) string {
			ret, _ := json.Marshal(data)
			return string(ret)
		},
	})
	var err error

	tpl, err = tpl.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	thsPage, err := gs.sdk.Things("123", 0, 100, "")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
		Things       []sdk.Thing
	}{
		"things",
		thsPage.Things,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "things", data); err != nil {
		println(err.Error())
	}

	return btpl.Bytes(), nil
}

func (gs *uiService) ViewThing(ctx context.Context, token, id string) ([]byte, error) {
	tpl := template.New("things")
	tpl = tpl.Funcs(template.FuncMap{
		"toJSON": func(data map[string]interface{}) string {
			ret, _ := json.Marshal(data)
			return string(ret)
		},
	})
	var err error
	tpl, err = tpl.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	thing, err := gs.sdk.Thing(id, "123")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
		ID           string
		Thing        sdk.Thing
	}{
		"things",
		id,
		thing,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "thing", data); err != nil {
		println(err.Error())
	}
	return btpl.Bytes(), nil
}

func (gs *uiService) UpdateThing(ctx context.Context, token, id string, thing sdk.Thing) ([]byte, error) {
	if err := gs.sdk.UpdateThing(thing, "123"); err != nil {
		return []byte{}, err
	}
	return gs.ViewThing(ctx, "123", id)
}

func (gs *uiService) RemoveThing(ctx context.Context, token, id string) ([]byte, error) {
	err := gs.sdk.DeleteThing(id, "123")
	if err != nil {
		return []byte{}, err
	}
	return []byte{}, nil
}

func (gs *uiService) CreateChannels(ctx context.Context, token string, channels ...sdk.Channel) ([]byte, error) {
	for i := range channels {
		_, err := gs.sdk.CreateChannel(channels[i], "123")
		if err != nil {
			return []byte{}, err
		}
	}
	return gs.ListChannels(ctx, "123")
}

func (gs *uiService) ViewChannel(ctx context.Context, token, id string) ([]byte, error) {
	tpl := template.New("channels")
	tpl = tpl.Funcs(template.FuncMap{
		"toJSON": func(data map[string]interface{}) string {
			ret, _ := json.Marshal(data)
			return string(ret)
		},
	})
	var err error
	tpl, err = tpl.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	channel, err := gs.sdk.Channel(id, "123")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
		ID           string
		Channel      sdk.Channel
	}{
		"channels",
		id,
		channel,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "channel", data); err != nil {
		println(err.Error())
	}
	return btpl.Bytes(), nil
}

func (gs *uiService) UpdateChannel(ctx context.Context, token, id string, channel sdk.Channel) ([]byte, error) {
	if err := gs.sdk.UpdateChannel(channel, "123"); err != nil {
		return []byte{}, err
	}
	return gs.ViewChannel(ctx, "123", id)
}

func (gs *uiService) ListChannels(ctx context.Context, token string) ([]byte, error) {
	tpl := template.New("channels")
	tpl = tpl.Funcs(template.FuncMap{
		"toJSON": func(data map[string]interface{}) string {
			ret, _ := json.Marshal(data)
			return string(ret)
		},
	})
	var err error

	tpl, err = tpl.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	chsPage, err := gs.sdk.Channels("123", 0, 100, "")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
		Channels     []sdk.Channel
	}{
		"channels",
		chsPage.Channels,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "channels", data); err != nil {
		println(err.Error())
	}

	return btpl.Bytes(), nil
}

func (gs *uiService) RemoveChannel(ctx context.Context, token, id string) ([]byte, error) {
	err := gs.sdk.DeleteChannel(id, "123")
	if err != nil {
		return []byte{}, err
	}
	return gs.ListChannels(ctx, "123")
}

func (gs *uiService) CreateGroups(ctx context.Context, token string, groups ...sdk.Group) ([]byte, error) {
	for i := range groups {
		_, err := gs.sdk.CreateGroup(groups[i], "123")
		if err != nil {
			return []byte{}, err
		}
	}
	return gs.ListGroups(ctx, "123")
}

func (gs *uiService) ListGroups(ctx context.Context, token string) ([]byte, error) {
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	grpsPage, err := gs.sdk.Groups(0, 100, "123")
	if err != nil {
		return []byte{}, err
	}

	data := struct {
		NavbarActive string
		Groups       []auth.Group
	}{
		"groups",
		grpsPage.Groups,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "groups", data); err != nil {
		println(err.Error())
	}

	return btpl.Bytes(), nil
}

func (gs *uiService) ViewGroup(ctx context.Context, token, id string) ([]byte, error) {
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	group, err := gs.sdk.Group(id, "123")
	if err != nil {
		return []byte{}, err
	}

	j, err := json.Marshal(group)
	if err != nil {
		return []byte{}, err
	}

	m := make(map[string]interface{})
	json.Unmarshal(j, &m)

	data := struct {
		NavbarActive string
		ID           string
		JSONGroup    map[string]interface{}
	}{
		"groups",
		id,
		m,
	}
	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "group", data); err != nil {
		println(err.Error())
	}
	return btpl.Bytes(), nil
}

func (gs *uiService) UpdateGroup(ctx context.Context, token, id string, group sdk.Group) ([]byte, error) {
	if err := gs.sdk.UpdateGroup(group, "123"); err != nil {
		return []byte{}, err
	}
	return gs.ViewGroup(ctx, "123", id)
}

func (gs *uiService) RemoveGroup(ctx context.Context, token, id string) ([]byte, error) {
	err := gs.sdk.DeleteGroup(id, "123")
	if err != nil {
		return []byte{}, err
	}
	return []byte{}, nil
}
