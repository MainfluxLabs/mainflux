// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/url"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/gofrs/uuid"
)

const (
	maxNameSize = 1024
	formatJSON  = "json"
	formatSenML = "senml"
)

var (
	ErrInvalidUrl    = errors.New("missing or invalid url")
	ErrInvalidFormat = errors.New("invalid format")
)

type apiReq interface {
	validate() error
}

type createWebhookReq struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Url    string `json:"url"`
}

type createWebhooksReq struct {
	Token    string             `json:"token"`
	ThingID  string             `json:"thing_id"`
	Webhooks []createWebhookReq `json:"webhooks"`
}

func (req createWebhooksReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	if err := validateUUID(req.ThingID); err != nil {
		return err
	}

	if len(req.Webhooks) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, wh := range req.Webhooks {
		if err := wh.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req createWebhookReq) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.Format != formatJSON && req.Format != formatSenML {
		return ErrInvalidFormat
	}

	_, err := url.ParseRequestURI(req.Url)
	if err != nil {
		return ErrInvalidUrl
	}

	return nil
}

type listWebhooksReq struct {
	Token   string `json:"token"`
	ThingID string `json:"thing_id"`
}

func (req *listWebhooksReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}
	if err := validateUUID(req.ThingID); err != nil {
		return err
	}
	return nil
}

func validateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return apiutil.ErrInvalidIDFormat
	}

	return nil
}
