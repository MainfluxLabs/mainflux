// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/api"
)

const (
	minLen       = 1
	maxLimitSize = 200
	maxNameSize  = 254
	// maxBatchSize bounds a create batch. Each script in it is compiled, and each
	// compile occupies a worker, so an unbounded batch starves every other caller.
	maxBatchSize = 32

	// maxRemoveBatch bounds a delete batch. The service authorizes each id with a
	// retrieve plus a gRPC call, so an unbounded list is cheap request
	// amplification against things and auth - a 4MB body holds ~100k uuids.
	maxRemoveBatch = 100
)

// MaxPayloadSize bounds the input payload accepted by a run request.
var MaxPayloadSize int64 = 1 << 20

// MaxBodySize bounds every other request body. Without it a create or update is
// buffered in full before MaxScriptSize is ever consulted.
var MaxBodySize int64 = 4 << 20

type script struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

func (s script) validate() error {
	if s.Name == "" || len(s.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if s.Source == "" {
		return errors.ErrMalformedEntity
	}

	if len(s.Source) > pyscripts.MaxScriptSize {
		return pyscripts.ErrScriptSize
	}

	return nil
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

	if len(req.Scripts) > maxBatchSize {
		return apiutil.ErrLimitSize
	}

	for _, s := range req.Scripts {
		if err := s.validate(); err != nil {
			return err
		}
	}

	return nil
}

type listScriptsByGroupReq struct {
	token        string
	groupID      string
	pageMetadata pyscripts.PageMetadata
}

func (req listScriptsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return api.ValidateScriptPageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
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
	token       string
	id          string
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

func (req updateScriptReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingScriptID
	}

	return script{Name: req.Name, Description: req.Description, Source: req.Source}.validate()
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

	if len(req.ScriptIDs) > maxRemoveBatch {
		return apiutil.ErrLimitSize
	}

	for _, id := range req.ScriptIDs {
		if id == "" {
			return apiutil.ErrMissingScriptID
		}
	}

	return nil
}

type runScriptReq struct {
	token       string
	id          string
	Payload     map[string]any `json:"payload"`
	Subtopic    string         `json:"subtopic,omitempty"`
	Created     int64          `json:"created,omitempty"`
	PublisherID string         `json:"publisher_id,omitempty"`
}

func (req runScriptReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingScriptID
	}

	return nil
}

type listRunsByScriptReq struct {
	token        string
	scriptID     string
	pageMetadata pyscripts.PageMetadata
}

func (req listRunsByScriptReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.scriptID == "" {
		return apiutil.ErrMissingScriptID
	}

	return api.ValidateRunPageMetadata(req.pageMetadata, maxLimitSize)
}

type removeRunsReq struct {
	token  string
	RunIDs []string `json:"run_ids"`
}

func (req removeRunsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.RunIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	if len(req.RunIDs) > maxRemoveBatch {
		return apiutil.ErrLimitSize
	}

	for _, id := range req.RunIDs {
		if id == "" {
			return apiutil.ErrMissingScriptRunID
		}
	}

	return nil
}
