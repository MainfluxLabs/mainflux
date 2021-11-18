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
	"fmt"
	"html/template"

	"github.com/mainflux/mainflux"
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
	ListThings(ctx context.Context, token string) ([]byte, error)
	UpdateThing(ctx context.Context, token string, thing sdk.Thing) ([]byte, error)
	RemoveThing(ctx context.Context, token, id string) ([]byte, error)
	CreateChannels(ctx context.Context, token string, channels ...sdk.Channel) ([]byte, error)
	ViewChannel(ctx context.Context, token, id string) ([]byte, error)
	UpdateChannel(ctx context.Context, token, id string, channel sdk.Channel) ([]byte, error)
	ListChannels(ctx context.Context, token string) ([]byte, error)
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
	tpl, err := template.ParseGlob(templateDir + "/*")
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
		fmt.Println(things[i])
		_, err := gs.sdk.CreateThing(things[i], "123")
		if err != nil {
			return []byte{}, err
		}
	}

	return gs.ListThings(ctx, "123")
}

func (gs *uiService) ListThings(ctx context.Context, token string) ([]byte, error) {
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	thsPage, err := gs.sdk.Things("123", 0, 100, "")
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(thsPage.Things)

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
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	thing, err := gs.sdk.Thing(id, "123")
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(thing)

	j, err := json.Marshal(thing)
	fmt.Println(string(j))
	if err != nil {
		return []byte{}, err
	}

	m := make(map[string]interface{})
	json.Unmarshal(j, &m)

	data := struct {
		NavbarActive string
		JSONThing    map[string]interface{}
	}{
		"things",
		m,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "thing", data); err != nil {
		println(err.Error())
	}
	fmt.Println(btpl.String())
	return btpl.Bytes(), nil
}

func (gs *uiService) UpdateThing(ctx context.Context, token string, thing sdk.Thing) ([]byte, error) {
	_, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	return gs.ListThings(ctx, "123")
}

func (gs *uiService) RemoveThing(ctx context.Context, token, id string) ([]byte, error) {
	_, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	// if err := gs.thingCache.Remove(ctx, tpl); err != nil {
	// 	return err
	// }
	return gs.ListChannels(ctx, "123")
}

func (gs *uiService) CreateChannels(ctx context.Context, token string, channels ...sdk.Channel) ([]byte, error) {
	fmt.Println("assss")
	for i := range channels {
		fmt.Println(channels[i])
		_, err := gs.sdk.CreateChannel(channels[i], "123")
		if err != nil {
			return []byte{}, err
		}
	}

	return gs.ListChannels(ctx, "123")
}

func (gs *uiService) ViewChannel(ctx context.Context, token, id string) ([]byte, error) {
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}
	channel, err := gs.sdk.Channel(id, "123")
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(channel)

	j, err := json.Marshal(channel)
	fmt.Println(string(j))
	if err != nil {
		return []byte{}, err
	}

	m := make(map[string]interface{})
	json.Unmarshal(j, &m)

	data := struct {
		NavbarActive string
		JSONChannel  map[string]interface{}
	}{
		"channels",
		m,
	}

	var btpl bytes.Buffer
	if err := tpl.ExecuteTemplate(&btpl, "channel", data); err != nil {
		println(err.Error())
	}
	fmt.Println(btpl.String())
	return btpl.Bytes(), nil
}

func (gs *uiService) UpdateChannel(ctx context.Context, token, id string, channel sdk.Channel) ([]byte, error) {
	if err := gs.sdk.UpdateChannel(channel, "123"); err != nil {
		return []byte{}, err
	}
	return gs.ViewChannel(ctx, id, "123")
}

func (gs *uiService) ListChannels(ctx context.Context, token string) ([]byte, error) {
	tpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return []byte{}, err
	}

	chsPage, err := gs.sdk.Channels("123", 0, 100, "")
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(chsPage.Channels)

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
