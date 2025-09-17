// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

const maxLimitSize = 1000

type listSenMLMessagesReq struct {
	token    string
	key      string
	pageMeta readers.SenMLPageMetadata
}

func (req listSenMLMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	if err := validateAggregation(req.pageMeta.AggType); err != nil {
		return err
	}

	return nil
}

type listJSONMessagesReq struct {
	token    string
	key      string
	pageMeta readers.JSONPageMetadata
}

func (req listJSONMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if err := validateAggregation(req.pageMeta.AggType); err != nil {
		return err
	}

	return nil
}

type backupSenMLMessagesReq struct {
	token         string
	convertFormat string
	pageMeta      readers.SenMLPageMetadata
}

func (req backupSenMLMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	if err := validateAggregation(req.pageMeta.AggType); err != nil {
		return err
	}

	return nil
}

type backupJSONMessagesReq struct {
	token         string
	convertFormat string
	pageMeta      readers.JSONPageMetadata
}

func (req backupJSONMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	if err := validateAggregation(req.pageMeta.AggType); err != nil {
		return err
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

type deleteSenMLMessagesReq struct {
	token    string
	key      string
	pageMeta readers.SenMLPageMetadata
}

func (req deleteSenMLMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type deleteJSONMessagesReq struct {
	token    string
	key      string
	pageMeta readers.JSONPageMetadata
}

func (req deleteJSONMessagesReq) validate() error {
	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

func validateAggregation(aggType string) error {
	if aggType == "" {
		return nil
	}

	switch aggType {
	case readers.AggregationMin, readers.AggregationMax, readers.AggregationAvg, readers.AggregationCount:
		return nil
	default:
		return apiutil.ErrInvalidAggType
	}
}
