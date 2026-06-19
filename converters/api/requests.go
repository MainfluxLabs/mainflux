// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

type convertCSVReq struct {
	csvLines [][]string
	key      domain.ThingKey
	to       string // "senml" or "json"
}

func (req convertCSVReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	if len(req.csvLines) < 1 || len(req.csvLines[0]) < 2 {
		return apiutil.ErrEmptyList
	}

	if req.to != "senml" && req.to != "json" {
		return apiutil.ErrInvalidQueryParams
	}

	return nil
}

type convertJSONReq struct {
	records []map[string]any
	key     domain.ThingKey
	to      string // "senml" or "json"
}

func (req convertJSONReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	if len(req.records) == 0 {
		return apiutil.ErrEmptyList
	}

	if req.to != "senml" && req.to != "json" {
		return apiutil.ErrInvalidQueryParams
	}

	return nil
}
