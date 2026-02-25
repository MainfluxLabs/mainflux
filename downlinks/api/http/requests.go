// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/url"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	minLen       = 1
	maxLimitSize = 200
	maxNameSize  = 254
	maxParamSize = 64
)

var (
	ErrMissingID             = errors.New("missing downlink id")
	ErrInvalidURL            = errors.New("missing or invalid url")
	ErrInvalidScheduler      = errors.New("missing or invalid scheduler")
	ErrMissingFilterFormat   = errors.New("missing time filter format")
	ErrInvalidFilterParam    = errors.New("invalid time filter param")
	ErrInvalidFilterInterval = errors.New("invalid time filter interval")
	ErrInvalidFilterValue    = errors.New("invalid time filter value")
)

type downlink struct {
	Name       string               `json:"name"`
	Url        string               `json:"url"`
	Method     string               `json:"method"`
	Payload    string               `json:"payload,omitempty"`
	Headers    map[string]string    `json:"headers,omitempty"`
	Scheduler  cron.Scheduler       `json:"scheduler"`
	TimeFilter downlinks.TimeFilter `json:"time_filter"`
	Metadata   map[string]any       `json:"metadata,omitempty"`
}

type createDownlinksReq struct {
	token     string
	thingID   string
	Downlinks []downlink `json:"downlinks"`
}

func (req createDownlinksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.Downlinks) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, dl := range req.Downlinks {
		if err := dl.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req downlink) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if _, err := url.ParseRequestURI(req.Url); err != nil {
		return ErrInvalidURL
	}

	if !req.Scheduler.IsValid() {
		return ErrInvalidScheduler
	}

	if req.TimeFilter.StartParam != "" && req.TimeFilter.EndParam != "" {
		switch req.TimeFilter.Interval {
		case downlinks.MinuteInterval, downlinks.HourInterval, downlinks.DayInterval:
		default:
			return ErrInvalidFilterInterval
		}

		if req.TimeFilter.Value == 0 {
			return ErrInvalidFilterValue
		}

		if len(req.TimeFilter.StartParam) > maxParamSize || len(req.TimeFilter.EndParam) > maxParamSize {
			return ErrInvalidFilterParam
		}

		if req.TimeFilter.Format == "" {
			return ErrMissingFilterFormat
		}
	}

	return nil
}

type listThingDownlinksReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listThingDownlinksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, downlinks.AllowedOrders)
}

type listDownlinksReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listDownlinksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, downlinks.AllowedOrders)
}

type downlinkReq struct {
	token string
	id    string
}

func (req *downlinkReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return ErrMissingID
	}
	return nil
}

type updateDownlinkReq struct {
	token string
	id    string
	downlink
}

func (req updateDownlinkReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return ErrMissingID
	}

	return req.downlink.validate()
}

type removeDownlinksReq struct {
	token       string
	DownlinkIDs []string `json:"downlink_ids,omitempty"`
}

func (req removeDownlinksReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.DownlinkIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, dlID := range req.DownlinkIDs {
		if dlID == "" {
			return ErrMissingID
		}
	}

	return nil
}
