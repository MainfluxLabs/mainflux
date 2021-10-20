// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ui contains the domain concept definitions needed to support
// Mainflux ui adapter service functionality.
package ui

import (
	"context"
	"fmt"
	"html/template"

	"github.com/mainflux/mainflux"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

const (
	templateDir = "ui/web/template"
)

// Service specifies coap service API.
type Service interface {
	Index(ctx context.Context, token string) (TemplateData, error)
	Things(ctx context.Context, token string) (TemplateData, error)
	Channels(ctx context.Context, token string) (TemplateData, error)
}

var _ Service = (*uiService)(nil)

type uiService struct {
	things mainflux.ThingsServiceClient
	sdk    sdk.SDK
}

type TemplateData struct {
	Template *template.Template
	Name     string
	Data     interface{}
}

// New instantiates the HTTP adapter implementation.
func New(things mainflux.ThingsServiceClient, sdk sdk.SDK) Service {
	return &uiService{
		things: things,
		sdk:    sdk,
	}
}

func (gs *uiService) Index(ctx context.Context, token string) (TemplateData, error) {
	tmpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return TemplateData{}, err
	}

	data := struct {
		NavbarActive string
	}{
		"dashboard",
	}

	return TemplateData{
		Template: tmpl,
		Name:     "index",
		Data:     data,
	}, nil
}

func (gs *uiService) Things(ctx context.Context, token string) (TemplateData, error) {
	tmpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return TemplateData{}, err
	}

	things, err := gs.sdk.Things("123", 0, 100, "")
	if err != nil {
		return TemplateData{}, err
	}
	fmt.Println(things)

	data := struct {
		NavbarActive string
	}{
		"things",
	}

	return TemplateData{
		Template: tmpl,
		Name:     "things",
		Data:     data,
	}, nil
}

func (gs *uiService) Channels(ctx context.Context, token string) (TemplateData, error) {
	tmpl, err := template.ParseGlob(templateDir + "/*")
	if err != nil {
		return TemplateData{}, err
	}

	data := struct {
		NavbarActive string
	}{
		"channels",
	}

	return TemplateData{
		Template: tmpl,
		Name:     "channels",
		Data:     data,
	}, nil
}
