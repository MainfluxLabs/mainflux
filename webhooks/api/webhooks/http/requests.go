// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"net/url"
)

const (
	maxNameSize = 1024
	formatJSON  = "json"
	formatSenML = "senml"
)

type apiReq interface {
	validate() error
}

type webhookReq struct {
	Name    string `json:"name"`
	Format  string `json:"format"`
	Url     string `json:"url"`
	Token   string `json:"token"`
	ThingID string `json:"thingID"`
}

func (req webhookReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.Format == "" {
		return errors.New("missing type of format")
	} else if req.Format != formatJSON && req.Format != formatSenML {
		return errors.New("invalid type of format")
	}

	_, err := url.ParseRequestURI(req.Url)
	if req.Url == "" || err != nil {
		return errors.New("missing or invalid url")
	}

	if req.ThingID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listWebhooksReq struct {
	Token   string `json:"token"`
	ThingID string `json:"thingID"`
}

func (req *listWebhooksReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}
	if req.ThingID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
