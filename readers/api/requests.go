// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

const (
	maxLimitSize = 1000
	noLimit      = -1
)

type listChannelMessagesReq struct {
	chanID   string
	token    string
	key      string
	pageMeta readers.PageMetadata
}

func (req listChannelMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMeta.Limit != noLimit {
		if req.pageMeta.Limit < 1 || req.pageMeta.Limit > maxLimitSize {
			return apiutil.ErrLimitSize
		}
	}

	if req.pageMeta.Offset < 0 {
		return apiutil.ErrOffsetSize
	}

	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	return nil
}
