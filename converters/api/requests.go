// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
)

type convertCSVReq struct {
	csvLines [][]string
	key      domainthings.ThingKey
}

func (req convertCSVReq) validate() error {
	if req.key.Value == "" {
		return apiutil.ErrBearerKey
	}

	if len(req.csvLines) < 1 || len(req.csvLines[0]) < 2 {
		return apiutil.ErrEmptyList
	}

	return nil
}
