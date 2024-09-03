// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	maxLimitSize = 100
	maxNameSize  = 254
	nameOrder    = "name"
	idOrder      = "id"
	ascDir       = "asc"
	descDir      = "desc"
)

type apiReq interface {
	validate() error
}

type notifierReq struct {
	token string
	id    string
}

func (req *notifierReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listNotifiersReq struct {
	token        string
	id           string
	pageMetadata things.PageMetadata
}

func (req listNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type createNotifierReq struct {
	Name     string                 `json:"name"`
	Contacts []string               `json:"contacts"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type createNotifiersReq struct {
	token     string
	groupID   string
	Notifiers []createNotifierReq `json:"notifiers"`
}

func (req createNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Notifiers) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, nf := range req.Notifiers {
		if err := nf.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req createNotifierReq) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if len(req.Contacts) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type updateNotifierReq struct {
	token    string
	id       string
	Name     string                 `json:"name"`
	Contacts []string               `json:"contacts"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateNotifierReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if len(req.Contacts) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type removeNotifiersReq struct {
	groupID     string
	token       string
	NotifierIDs []string `json:"notifier_ids,omitempty"`
}

func (req removeNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.NotifierIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, nfID := range req.NotifierIDs {
		if nfID == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}
