// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/things"
)

const maxLimitSize = 1000

type listSenMLMessagesReq struct {
	token    string
	thingKey things.ThingKey
	pageMeta readers.SenMLPageMetadata
}

func (req listSenMLMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if err := validateDir(req.pageMeta.Dir); err != nil {
		return err
	}

	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggInterval, req.pageMeta.AggValue); err != nil {
		return err
	}

	return nil
}

type listJSONMessagesReq struct {
	token    string
	thingKey things.ThingKey
	pageMeta readers.JSONPageMetadata
}

func (req listJSONMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if err := validateDir(req.pageMeta.Dir); err != nil {
		return err
	}

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggInterval, req.pageMeta.AggValue); err != nil {
		return err
	}

	return nil
}

type backupSenMLMessagesReq struct {
	token         string
	convertFormat string
	timeFormat    string
	pageMeta      readers.SenMLPageMetadata
}

func (req backupSenMLMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggInterval, req.pageMeta.AggValue); err != nil {
		return err
	}

	if err := validateDir(req.pageMeta.Dir); err != nil {
		return err
	}

	return nil
}

type backupJSONMessagesReq struct {
	token         string
	convertFormat string
	timeFormat    string
	pageMeta      readers.JSONPageMetadata
}

func (req backupJSONMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggInterval, req.pageMeta.AggValue); err != nil {
		return err
	}

	if err := validateDir(req.pageMeta.Dir); err != nil {
		return err
	}

	return nil
}

type restoreMessagesReq struct {
	token    string
	fileType string
	Messages []byte
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

type deleteAllSenMLMessagesReq struct {
	token    string
	thingKey things.ThingKey
	pageMeta readers.SenMLPageMetadata
}

func (req deleteAllSenMLMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}

	return nil
}

type deleteAllJSONMessagesReq struct {
	token    string
	thingKey things.ThingKey
	pageMeta readers.JSONPageMetadata
}

func (req deleteAllJSONMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}
	return nil
}

type deleteJSONMessagesReq struct {
	token    string
	pageMeta readers.JSONPageMetadata
}

func (req deleteJSONMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Publisher == "" {
		return apiutil.ErrMissingPublisherID
	}

	return nil
}

type deleteSenMLMessagesReq struct {
	token    string
	pageMeta readers.SenMLPageMetadata
}

func (req deleteSenMLMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMeta.Publisher == "" {
		return apiutil.ErrMissingPublisherID
	}

	return nil
}

func validateAggregation(aggType, aggInterval string, aggValue uint64) error {
	if aggInterval == "" || aggType == "" {
		return nil
	}

	if !isValidAggInterval(aggInterval, aggValue) {
		return apiutil.ErrInvalidAggInterval
	}

	switch aggType {
	case readers.AggregationMin, readers.AggregationMax, readers.AggregationAvg, readers.AggregationCount:
		return nil
	default:
		return apiutil.ErrInvalidAggType
	}
}

func validateDir(dir string) error {
	if dir == "" || dir == apiutil.AscDir || dir == apiutil.DescDir {
		return nil
	}
	return apiutil.ErrInvalidDirection
}

func isValidAggInterval(aggInterval string, aggValue uint64) bool {
	var maxValue uint64

	switch aggInterval {
	case "minute":
		maxValue = 60
	case "hour":
		maxValue = 24
	case "day":
		maxValue = 31
	case "week":
		maxValue = 52
	case "month":
		maxValue = 12
	case "year":
		maxValue = 10
	default:
		return false
	}

	return aggValue <= maxValue
}
