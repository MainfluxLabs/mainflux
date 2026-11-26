// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	mfreaders "github.com/MainfluxLabs/mainflux/pkg/readers"
)

type listJSONMessagesReq struct {
	thingKey domain.ThingKey
	pm       domain.JSONPageMetadata
}

func (req listJSONMessagesReq) validate() error {
	if req.pm.Key != "" {
		if _, err := mfreaders.ParseKey(req.pm.Key); err != nil {
			return apiutil.ErrInvalidQueryParams
		}
	}

	if len(req.pm.KeyValues) > mfreaders.MaxKeyValues {
		return apiutil.ErrInvalidQueryParams
	}

	hasValue := req.pm.Value != ""
	for _, kv := range req.pm.KeyValues {
		if _, err := mfreaders.ParseKey(kv.Key); err != nil {
			return apiutil.ErrInvalidQueryParams
		}

		if kv.Value != "" {
			hasValue = true
		}
	}

	if !hasValue {
		if req.pm.Comparator != "" {
			return apiutil.ErrInvalidQueryParams
		}

		return nil
	}

	switch req.pm.Comparator {
	case "", mfreaders.EqualKey, mfreaders.StartsWithKey, mfreaders.ContainsKey:
		return nil
	default:
		return apiutil.ErrInvalidComparator
	}
}

type listSenMLMessagesReq struct {
	thingKey domain.ThingKey
	pm       domain.SenMLPageMetadata
}
