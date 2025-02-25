// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
	nameOrder    = "name"
	idOrder      = "id"
	ascDir       = "asc"
	descDir      = "desc"
)

type createThingReq struct {
	ProfileID string                 `json:"profile_id"`
	Name      string                 `json:"name,omitempty"`
	Key       string                 `json:"key,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type createThingsReq struct {
	token   string
	groupID string
	Things  []createThingReq
}

func (req createThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Things) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, thing := range req.Things {
		if thing.ProfileID == "" {
			return apiutil.ErrMissingID
		}
		if thing.ID != "" {
			if err := validateUUID(thing.ID); err != nil {
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
		return apiutil.ErrMissingID
	}

	if req.ProfileID == "" {
		return apiutil.ErrMissingID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateReq struct {
	ProfileID string                 `json:"profile_id"`
	Name      string                 `json:"name,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type updateThingsReq struct {
	token  string
	Things []updateReq
}

func (req updateThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	for _, thing := range req.Things {
		if thing.ProfileID == "" {
			return apiutil.ErrMissingID
		}

		if thing.ID != "" {
			return apiutil.ErrMissingID
		}

		if thing.Name == "" || len(thing.Name) > maxNameSize {
			return apiutil.ErrNameSize
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
		return apiutil.ErrMissingID
	}

	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type viewMetadataReq struct {
	key string
}

func (req viewMetadataReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

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
			if err := validateUUID(profile.ID); err != nil {
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
		return apiutil.ErrMissingID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
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
			return apiutil.ErrMissingID
		}
	}

	return nil
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
			return apiutil.ErrMissingID
		}
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
		return apiutil.ErrMissingID
	}

	return nil
}

type listResourcesReq struct {
	token        string
	pageMetadata things.PageMetadata
}

func (req *listResourcesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return apiutil.ErrNameSize
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

type listByIDReq struct {
	token        string
	id           string
	pageMetadata things.PageMetadata
}

func (req listByIDReq) validate() error {
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

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreThingReq struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreProfileReq struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreGroupReq struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type restoreReq struct {
	token    string
	Things   []restoreThingReq   `json:"things"`
	Profiles []restoreProfileReq `json:"profiles"`
	Groups   []restoreGroupReq   `json:"groups"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Groups) == 0 && len(req.Things) == 0 && len(req.Profiles) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type createGroupReq struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type createGroupsReq struct {
	token  string
	orgID  string
	Groups []createGroupReq
}

func (req createGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if len(req.Groups) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, group := range req.Groups {
		if group.Name == "" || len(group.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type updateGroupReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

func validateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return apiutil.ErrInvalidIDFormat
	}

	return nil
}

type removeGroupsReq struct {
	token    string
	GroupIDs []string `json:"group_ids,omitempty"`
}

func (req removeGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.GroupIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, groupID := range req.GroupIDs {
		if groupID == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}

type identifyReq struct {
	Token string `json:"token"`
}

func (req identifyReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type groupRolesReq struct {
	token        string
	groupID      string
	GroupMembers []groupMember `json:"group_members"`
}

func (req groupRolesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.GroupMembers) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, gm := range req.GroupMembers {
		if gm.Role != auth.Admin && gm.Role != things.Viewer && gm.Role != things.Editor {
			return apiutil.ErrInvalidRole
		}

		if gm.ID == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}

type listGroupsReq struct {
	token        string
	orgID        string
	pageMetadata things.PageMetadata
}

func (req listGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type removeGroupRolesReq struct {
	token     string
	groupID   string
	MemberIDs []string `json:"member_ids"`
}

func (req removeGroupRolesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.MemberIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, id := range req.MemberIDs {
		if id == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}
