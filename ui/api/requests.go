// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/ui"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
	nameOrder    = "name"
	idOrder      = "id"
	ascDir       = "asc"
	descDir      = "desc"
)

type indexReq struct {
	token string
}

type createThingsReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req createThingsReq) validate() error {
	if req.token == "" {
		return ui.ErrUnauthorizedAccess
	}

	if len(req.Name) > maxNameSize {
		return ui.ErrMalformedEntity
	}

	return nil
}

type listThingsReq struct {
	token string
}

type viewResourceReq struct {
	token string
	id    string
}

func (req viewResourceReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type updateThingReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type createChannelsReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req createChannelsReq) validate() error {
	if req.token == "" {
		return ui.ErrUnauthorizedAccess
	}

	if len(req.Name) > maxNameSize {
		return ui.ErrMalformedEntity
	}

	return nil
}

type updateChannelReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateChannelReq) validate() error {

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type listChannelsReq struct {
	token string
}

type createGroupsReq struct {
	token       string                 `json:"token,omitempty"`
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req createGroupsReq) validate() error {
	if req.ID == "" {
		return ui.ErrUnauthorizedAccess
	}

	if len(req.Name) > maxNameSize {
		return ui.ErrMalformedEntity
	}

	return nil
}

type listGroupsReq struct {
	token string
}

type updateGroupReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {

	if req.token == "" {
		return things.ErrMalformedEntity
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}
