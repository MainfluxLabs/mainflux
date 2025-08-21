// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

const maxLimitSize = 1000

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

	if req.pageMeta.AggType != "" {
		switch req.pageMeta.AggType {
		case readers.AggregationMin, readers.AggregationMax, readers.AggregationAvg, readers.AggregationCount:
		default:
			return apiutil.ErrInvalidAggType
		}
	}

	return nil
}

type backupMessagesReq struct {
	token         string
	convertFormat string
	messageFormat string
	pageMeta      readers.PageMetadata
}

func (req backupMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	return nil
}

type restoreMessagesReq struct {
	token         string
	fileType      string
	messageFormat string
	Messages      []byte
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

type deleteMessagesReq struct {
	token    string
	key      string
	pageMeta readers.PageMetadata
}

func (req deleteMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}
