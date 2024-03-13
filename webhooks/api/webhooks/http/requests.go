// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"net/url"
	"strings"
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
	Name   string `json:"name"`
	Format string `json:"format"`
	Url    string `json:"url"`
	Token  string `json:"token"`
}

func (req webhookReq) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	f := strings.ToLower(req.Format)
	if f == "" {
		return errors.New("missing type of Format")
	} else if f != formatJSON && f != formatSenML {
		return errors.New("invalid type of Format")
	}

	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	_, err := url.ParseRequestURI(req.Url)
	if req.Url == "" || err != nil {
		return errors.New("missing or invalid url")
	}

	return nil
}

type listWebhooksReq struct {
	token string `json:"token"`
}

func (req *listWebhooksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}
