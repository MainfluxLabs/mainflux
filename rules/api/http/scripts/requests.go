// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package scripts

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

const (
	minLen        = 1
	maxLimitSize  = 200
	maxNameSize   = 254
	maxScriptSize = 65_535
)

type script struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Script      string `json:"script"`
}

type createScriptsReq struct {
	token   string
	groupID string
	Scripts []script `json:"scripts"`
}

func (req createScriptsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Scripts) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, s := range req.Scripts {
		if s.Name == "" || len(s.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}

		if s.Script == "" {
			return apiutil.ErrMalformedEntity
		}

		if len(s.Script) > maxScriptSize {
			return rules.ErrScriptSize
		}
	}

	return nil
}

type listScriptsByThingReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listScriptsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type listScriptsByGroupReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listScriptsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type scriptReq struct {
	token string
	id    string
}

func (req scriptReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingScriptID
	}

	return nil
}

type updateScriptReq struct {
	token string
	id    string
	script
}

func (req updateScriptReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingScriptID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.Script == "" {
		return apiutil.ErrMalformedEntity
	}

	if len(req.Script) > maxScriptSize {
		return rules.ErrScriptSize
	}

	return nil
}

type removeScriptsReq struct {
	token     string
	ScriptIDs []string `json:"script_ids"`
}

func (req removeScriptsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ScriptIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, id := range req.ScriptIDs {
		if id == "" {
			return apiutil.ErrMissingScriptID
		}
	}

	return nil
}

type thingScriptsReq struct {
	token     string
	thingID   string
	ScriptIDs []string `json:"script_ids"`
}

func (req thingScriptsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.ScriptIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, id := range req.ScriptIDs {
		if id == "" {
			return apiutil.ErrMissingScriptID
		}
	}

	return nil
}

type listScriptRunsByThingReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listScriptRunsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type removeScriptRunsReq struct {
	token        string
	ScriptRunIDs []string `json:"script_run_ids"`
}

func (req removeScriptRunsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ScriptRunIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, id := range req.ScriptRunIDs {
		if id == "" {
			return apiutil.ErrMissingScriptRunID
		}
	}

	return nil
}
