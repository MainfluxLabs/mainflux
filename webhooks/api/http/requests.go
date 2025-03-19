// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/url"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	minLen       = 1
	maxLimitSize = 100
	maxNameSize  = 254
)

var ErrInvalidUrl = errors.New("missing or invalid url")

type createWebhookReq struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Url      string                 `json:"url"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type createWebhooksReq struct {
	token    string
	thingID  string
	Webhooks []createWebhookReq `json:"webhooks"`
}

func (req createWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.Webhooks) < minLen {
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
	if req.id == "" {
		return apiutil.ErrMissingWebhookID
	}
	return nil
}

type listWebhooksReq struct {
	token        string
	id           string
	pageMetadata apiutil.PageMetadata
}

func (req listWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != apiutil.NameOrder && req.pageMetadata.Order != apiutil.IDOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != apiutil.AscDir && req.pageMetadata.Dir != apiutil.DescDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type updateWebhookReq struct {
	token    string
	id       string
	Name     string                 `json:"name"`
	Url      string                 `json:"url"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateWebhookReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingWebhookID
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
	token      string
	WebhookIDs []string `json:"webhook_ids,omitempty"`
}

func (req removeWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.WebhookIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, whID := range req.WebhookIDs {
		if whID == "" {
			return apiutil.ErrMissingWebhookID
		}
	}

	return nil
}
