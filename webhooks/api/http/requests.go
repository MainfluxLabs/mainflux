// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/url"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/gofrs/uuid"
)

const maxNameSize = 1024

var ErrInvalidUrl = errors.New("missing or invalid url")

type apiReq interface {
	validate() error
}

type createWebhookReq struct {
	ID      string            `json:"id,omitempty"`
	Name    string            `json:"name"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type createWebhooksReq struct {
	token    string
	groupID  string
	Webhooks []createWebhookReq `json:"webhooks"`
}

func (req createWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if err := validateUUID(req.groupID); err != nil {
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

	_, err := url.ParseRequestURI(req.Url)
	if err != nil {
		return ErrInvalidUrl
	}

	return nil
}

type webhookReq struct {
	token string
	id    string
}

func (req *webhookReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if err := validateUUID(req.id); err != nil {
		return err
	}
	return nil
}

type updateWebhookReq struct {
	token   string
	id      string
	Name    string            `json:"name"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

func (req updateWebhookReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	_, err := url.ParseRequestURI(req.Url)
	if err != nil {
		return ErrInvalidUrl
	}

	return nil
}

type removeWebhooksReq struct {
	groupID    string
	token      string
	WebhookIDs []string `json:"webhook_ids,omitempty"`
}

func (req removeWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.WebhookIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, whID := range req.WebhookIDs {
		if whID == "" {
			return apiutil.ErrMissingID
		}
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
