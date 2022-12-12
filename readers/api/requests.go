// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

const (
	maxLimitSize = 1000
)

type listChannelMessagesReq struct {
	chanID   string
	token    string
	key      string
	pageMeta readers.PageMetadata
}

func (req listChannelMessagesReq) validateWithChannel() error {
	if req.chanID == "" {
		return apiutil.ErrMissingID
	}

	return validate(req.chanID, req.token, req.key, req.pageMeta)
}

type listMessagesReq struct {
	token    string
	key      string
	pageMeta readers.PageMetadata
}

func (req listMessagesReq) validateWithNoChanel() error {
	return validate("", req.token, req.key, req.pageMeta)
}

func validate(chanID, token, key string, pageMeta readers.PageMetadata) error {
	if token == "" && key == "" {
		return apiutil.ErrBearerToken
	}

	if pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if pageMeta.Offset < 0 {
		return apiutil.ErrOffsetSize
	}

	if pageMeta.Comparator != "" &&
		pageMeta.Comparator != readers.EqualKey &&
		pageMeta.Comparator != readers.LowerThanKey &&
		pageMeta.Comparator != readers.LowerThanEqualKey &&
		pageMeta.Comparator != readers.GreaterThanKey &&
		pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	return nil
}
