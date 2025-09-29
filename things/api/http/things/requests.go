// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things/api/http/memberships"
)

const (
	maxLimitSize = 200
	maxNameSize  = 1024
)

type createThingReq struct {
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
	token     string
	id        string
	ProfileID string                 `json:"profile_id"`
	Name      string                 `json:"name,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	if req.ProfileID == "" {
		return apiutil.ErrMissingProfileID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateMetadataReq struct {
	ID       string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

type updateKeyReq struct {
	token string
	id    string
	Key   string `json:"key"`
}

func (req updateKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type viewMetadataReq struct {
	apiutil.ThingKey
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

type backupByGroupReq struct {
	id    string
	token string
}

func (req backupByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}
	return nil
}

type backupByOrgReq struct {
	id    string
	token string
}

func (req backupByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingOrgID
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

type restoreThingsByGroupReq struct {
	id     string
	token  string
	Backup backupThings
}

func (req restoreThingsByGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Backup.Things) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type restoreThingsByOrgReq struct {
	id     string
	token  string
	Backup backupThings
}

func (req restoreThingsByOrgReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Backup.Things) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type restoreReq struct {
	token            string
	Things           backupThings                         `json:"things"`
	Profiles         []backupProfile                      `json:"profiles"`
	Groups           []backupGroup                        `json:"groups"`
	GroupMemberships []memberships.ViewGroupMembershipRes `json:"group_memberships"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	// FIXME: Why do we only validate only the existence of Things in the restore request?
	if len(req.Things.Things) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type identifyReq struct {
	apiutil.ThingKey
}

func (req identifyReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
	}

	return nil
}

type createExternalKeyReq struct {
	thingID string
	Key     string `json:"key"`
	token   string
}

func (req createExternalKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Key == "" {
		return apiutil.ErrMissingExternalThingKey
	}

	return nil
}

type listExternalKeysByThingReq struct {
	token   string
	thingID string
}

func (req listExternalKeysByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type removeExternalKeyReq struct {
	token string
	key   string
}

func (req removeExternalKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.key == "" {
		return apiutil.ErrMissingExternalThingKey
	}

	return nil
}
