// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

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

type exportSenMLMessagesReq struct {
	token         string
	convertFormat string
	timeFormat    string
	pageMeta      readers.SenMLPageMetadata
}

func (req exportSenMLMessagesReq) validate() error {
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

type exportJSONMessagesReq struct {
	token         string
	convertFormat string
	timeFormat    string
	pageMeta      readers.JSONPageMetadata
}

func (req exportJSONMessagesReq) validate() error {
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

type searchRequest struct {
	Type  string                     `json:"type"`
	JSON  *readers.JSONPageMetadata  `json:"json_params, omitempty"`
	SenML *readers.SenMLPageMetadata `json:"senml_params, omitempty"`
}

type searchMessagesReq struct {
	token    string
	thingKey things.ThingKey
	Searches []searchRequest `json:"searches"`
}

func (req searchMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Searches) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, s := range req.Searches {
		if s.Type != "json" && s.Type != "senml" {
			return apiutil.ErrInvalidQueryParams
		}

		if s.Type == "json" && s.JSON == nil {
			return apiutil.ErrMalformedEntity
		}
		if s.Type == "senml" && s.SenML == nil {
			return apiutil.ErrMalformedEntity
		}

		if s.Type == "json" {
			if s.JSON.Limit == 0 {
				s.JSON.Limit = apiutil.DefLimit
			}
			if s.JSON.Limit > maxLimitSize {
				return apiutil.ErrLimitSize
			}
			if err := validateDir(s.JSON.Dir); err != nil {
				return err
			}
			if err := validateAggregation(s.JSON.AggType, s.JSON.AggInterval, s.JSON.AggValue); err != nil {
				return err
			}
		}

		if s.Type == "senml" {
			if s.SenML.Limit == 0 {
				s.SenML.Limit = apiutil.DefLimit
			}
			if s.SenML.Limit > maxLimitSize {
				return apiutil.ErrLimitSize
			}
			if err := validateDir(s.SenML.Dir); err != nil {
				return err
			}
			if err := validateAggregation(s.SenML.AggType, s.SenML.AggInterval, s.SenML.AggValue); err != nil {
				return err
			}
		}
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
