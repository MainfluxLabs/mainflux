// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/messaging"
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
	token       string
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

type connectThingReq struct {
	token   string
	ChanID  string `json:"chan_id,omitempty"`
	ThingID string `json:"thing_id,omitempty"`
}

func (req connectThingReq) validate() error {
	// if req.token == "" {
	// 	return things.ErrUnauthorizedAccess
	// }

	if req.ChanID == "" || req.ThingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type connectChannelReq struct {
	token   string
	ThingID string `json:"thing_id,omitempty"`
	ChanID  string `json:"chan_id,omitempty"`
}

func (req connectChannelReq) validate() error {
	// if req.token == "" {
	// 	return things.ErrUnauthorizedAccess
	// }

	if req.ChanID == "" || req.ThingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type disconnectThingReq struct {
	token   string
	ChanID  string `json:"chan_id,omitempty"`
	ThingID string `json:"thing_id,omitempty"`
}

func (req disconnectThingReq) validate() error {
	// if req.token == "" {
	// 	return things.ErrUnauthorizedAccess
	// }

	if req.ChanID == "" || req.ThingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type disconnectChannelReq struct {
	token   string
	ThingID string `json:"thing_id,omitempty"`
	ChanID  string `json:"chan_id,omitempty"`
}

func (req disconnectChannelReq) validate() error {
	// if req.token == "" {
	// 	return things.ErrUnauthorizedAccess
	// }

	if req.ChanID == "" || req.ThingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type assignReq struct {
	token   string
	groupID string
	Type    string `json:"type,omitempty"`
	Member  string `json:"member"`
}

func (req assignReq) validate() error {
	if req.token == "" {
		return auth.ErrUnauthorizedAccess
	}

	if req.Type == "" || req.groupID == "" || req.Member == "" {
		return auth.ErrMalformedEntity
	}

	return nil
}

type unassignReq struct {
	assignReq
}

func (req unassignReq) validate() error {
	if req.token == "" {
		return auth.ErrUnauthorizedAccess
	}

	if req.groupID == "" || req.Member == "" {
		return auth.ErrMalformedEntity
	}

	return nil
}

type publishReq struct {
	msg      messaging.Message
	thingKey string
	token    string
}

type sendMessageReq struct {
	token string
}
