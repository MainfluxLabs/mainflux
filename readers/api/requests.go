// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"strings"

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

	if req.pageMeta.Comparator != "" &&
		req.pageMeta.Comparator != readers.EqualKey &&
		req.pageMeta.Comparator != readers.LowerThanKey &&
		req.pageMeta.Comparator != readers.LowerThanEqualKey &&
		req.pageMeta.Comparator != readers.GreaterThanKey &&
		req.pageMeta.Comparator != readers.GreaterThanEqualKey {
		return apiutil.ErrInvalidComparator
	}

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggValue, req.pageMeta.AggInterval); err != nil {
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

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggValue, req.pageMeta.AggInterval); err != nil {
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

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggValue, req.pageMeta.AggInterval); err != nil {
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

	if err := validateAggregation(req.pageMeta.AggType, req.pageMeta.AggValue, req.pageMeta.AggInterval); err != nil {
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
	thingKey things.ThingKey
	pageMeta readers.SenMLPageMetadata
}

func (req deleteSenMLMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}

	return nil
}

type deleteJSONMessagesReq struct {
	token    string
	thingKey things.ThingKey
	pageMeta readers.JSONPageMetadata
}

func (req deleteJSONMessagesReq) validate() error {
	err := req.thingKey.Validate()
	if req.token == "" && err != nil {
		return apiutil.ErrBearerToken
	}
	return nil
}

func validateAggregation(aggType string, aggIntervalValue int64, aggIntervalUnit string) error {
	maxValue := getAggIntervalLimit(aggIntervalUnit)
	if maxValue > 0 {
		if aggIntervalValue <= 0 {
			return apiutil.ErrInvalidAggInterval
		}

		if aggIntervalValue > maxValue {
			return apiutil.ErrInvalidAggInterval
		}
	}

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

func getAggIntervalLimit(unit string) int64 {
	normalizedUnit := strings.TrimSuffix(unit, "s")

	switch normalizedUnit {
	case "minute":
		return 60
	case "hour":
		return 24
	case "day":
		return 31
	case "month":
		return 12
	case "year":
		return 100
	default:
		return 0
	}
}
