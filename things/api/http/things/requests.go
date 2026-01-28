// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/api/http/memberships"
)

const (
	maxLimitSize = 200
	maxNameSize  = 1024
)

type createThingReq struct {
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key,omitempty"`
	ExternalKey string         `json:"external_key,omitempty"`
	ID          string         `json:"id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type createThingsReq struct {
	token     string
	profileID string
	Things    []createThingReq
}

func (req createThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.profileID == "" {
		return apiutil.ErrMissingProfileID
	}

	if len(req.Things) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, thing := range req.Things {
		if thing.ID != "" {
			if err := apiutil.ValidateUUID(thing.ID); err != nil {
				return err
			}
		}

		if thing.Name == "" || len(thing.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type updateThingReq struct {
	token    string
	id       string
	Key      string         `json:"key,omitempty"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateThingGroupAndProfileReq struct {
	token     string
	id        string
	ProfileID string `json:"profile_id"`
	GroupID   string `json:"group_id"`
}

func (req updateThingGroupAndProfileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	if req.ProfileID == "" {
		return apiutil.ErrMissingProfileID
	}

	if req.GroupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}

type updateMetadataReq struct {
	ID       string         `json:"id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type updateThingsMetadataReq struct {
	token  string
	Things []updateMetadataReq
}

func (req updateThingsMetadataReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	for _, thing := range req.Things {
		if thing.ID == "" {
			return apiutil.ErrMissingThingID
		}
	}

	return nil
}

type viewMetadataReq struct {
	things.ThingKey
}

func (req viewMetadataReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
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
		return apiutil.ErrMissingThingID
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

type listByProfileReq struct {
	id           string
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req listByProfileReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingProfileID
	}

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

type removeThingsReq struct {
	token    string
	ThingIDs []string `json:"thing_ids,omitempty"`
}

func (req removeThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ThingIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, thingID := range req.ThingIDs {
		if thingID == "" {
			return apiutil.ErrMissingThingID
		}
	}

	return nil
}

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreReq struct {
	token            string
	Things           []viewThingRes                       `json:"things"`
	Profiles         []backupProfile                      `json:"profiles"`
	Groups           []backupGroup                        `json:"groups"`
	GroupMemberships []memberships.ViewGroupMembershipRes `json:"group_memberships"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Things) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type identifyReq struct {
	things.ThingKey
}

func (req identifyReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
	}

	return nil
}

type updateExternalKeyReq struct {
	thingID string
	Key     string `json:"key"`
	token   string
}

func (req updateExternalKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Key == "" {
		return apiutil.ErrMissingExternalThingKey
	}

	return nil
}
