// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	minLen       = 1
	maxLimitSize = 100
	maxNameSize  = 254
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
		return apiutil.ErrMissingNotifierID
	}
	return nil
}

type listNotifiersReq struct {
	token        string
	id           string
	pageMetadata apiutil.PageMetadata
}

func (req listNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != apiutil.NameOrder && req.pageMetadata.Order != apiutil.IDOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != apiutil.AscDir && req.pageMetadata.Dir != apiutil.DescDir {
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

	if len(req.Notifiers) < minLen {
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

	if len(req.Contacts) < minLen {
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
		return apiutil.ErrMissingNotifierID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if len(req.Contacts) < minLen {
		return apiutil.ErrEmptyList
	}

	return nil
}

type removeNotifiersReq struct {
	token       string
	NotifierIDs []string `json:"notifier_ids,omitempty"`
}

func (req removeNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.NotifierIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, nfID := range req.NotifierIDs {
		if nfID == "" {
			return apiutil.ErrMissingNotifierID
		}
	}

	return nil
}
