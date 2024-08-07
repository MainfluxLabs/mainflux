// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
)

const maxLimitSize = 1000

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

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
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

type listAllMessagesReq struct {
	token    string
	key      string
	pageMeta readers.PageMetadata
}

func (req listAllMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
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

type restoreMessagesReq struct {
	token    string
	Messages []senml.Message `json:"messages"`
}

func (req restoreMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Messages) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
