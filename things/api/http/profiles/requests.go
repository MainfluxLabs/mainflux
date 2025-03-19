// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
)

type createProfileReq struct {
	Name     string                 `json:"name,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type createProfilesReq struct {
	token    string
	groupID  string
	Profiles []createProfileReq
}

func (req createProfilesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Profiles) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, profile := range req.Profiles {
		if profile.ID != "" {
			if err := apiutil.ValidateUUID(profile.ID); err != nil {
				return err
			}
		}

		if profile.Name == "" || len(profile.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type updateProfileReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateProfileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingProfileID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type viewByThingReq struct {
	token string
	id    string
}

func (req viewByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}

type resourceReq struct {
	token string
	id    string
}

func (req resourceReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingProfileID
	}

	return nil
}

type listReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req *listReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type listByGroupReq struct {
	id           string
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req listByGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type listByOrgReq struct {
	id           string
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req listByOrgReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type removeProfilesReq struct {
	token      string
	ProfileIDs []string `json:"profile_ids,omitempty"`
}

func (req removeProfilesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ProfileIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, profileID := range req.ProfileIDs {
		if profileID == "" {
			return apiutil.ErrMissingProfileID
		}
	}

	return nil
}
