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
	formatJSON  = "JSON"
	formatSenML = "SenML"
)

type apiReq interface {
	validate() error
}

type webhookReq struct {
	name   string `json:"name"`
	format string `json:"format"`
	url    string `json:"url"`
	token  string `json:"token"`
}

func (req webhookReq) validate() error {
	if req.name == "" || len(req.name) > maxNameSize {
		return errors.New("missing or invalid name ")
	}

	if req.format == "" {
		return errors.New("missing type of format")
	} else if req.format != formatJSON && req.format != formatSenML {
		return errors.New("invalid type of format")
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	_, err := url.ParseRequestURI(req.url)
	if req.url == "" || err != nil {
		return errors.New("missing or invalid url")
	}

	return nil
}
