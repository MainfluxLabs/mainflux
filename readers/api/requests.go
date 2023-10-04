// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
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

	if req.chanID == "" {
		return apiutil.ErrMissingID
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

type messageReq struct {
	Channel     string   `json:"channel,omitempty" db:"channel" bson:"channel"`
	Subtopic    string   `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	Protocol    string   `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Name        string   `json:"name,omitempty" db:"name" bson:"name,omitempty"`
	Unit        string   `json:"unit,omitempty" db:"unit" bson:"unit,omitempty"`
	Time        float64  `json:"time,omitempty" db:"time" bson:"time,omitempty"`
	UpdateTime  float64  `json:"update_time,omitempty" db:"update_time" bson:"update_time,omitempty"`
	Value       *float64 `json:"value,omitempty" db:"value" bson:"value,omitempty"`
	StringValue *string  `json:"string_value,omitempty" db:"string_value" bson:"string_value,omitempty"`
	DataValue   *string  `json:"data_value,omitempty" db:"data_value" bson:"data_value,omitempty"`
	BoolValue   *bool    `json:"bool_value,omitempty" db:"bool_value" bson:"bool_value,omitempty"`
	Sum         *float64 `json:"sum,omitempty" db:"sum" bson:"sum,omitempty"`
}

type restoreMessagesReq struct {
	token    string
	Messages []messageReq `json:"messages"`
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
