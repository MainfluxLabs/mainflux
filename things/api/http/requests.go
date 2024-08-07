// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"time"

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
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
		if thing.ID != "" {
			if err := validateUUID(thing.ID); err != nil {
				return err
			}
		}

		if len(thing.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
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
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
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

type createChannelReq struct {
	Name     string                 `json:"name,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type createChannelsReq struct {
	token    string
	groupID  string
	Channels []createChannelReq
}

func (req createChannelsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Channels) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, channel := range req.Channels {
		if channel.ID != "" {
			if err := validateUUID(channel.ID); err != nil {
				return err
			}
		}

		if len(channel.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type updateChannelReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Name) > maxNameSize {
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

type removeChannelsReq struct {
	token      string
	ChannelIDs []string `json:"channel_ids,omitempty"`
}

func (req removeChannelsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ChannelIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, channelID := range req.ChannelIDs {
		if channelID == "" {
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

type listByConnectionReq struct {
	token        string
	id           string
	pageMetadata things.PageMetadata
}

func (req listByConnectionReq) validate() error {
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

type connectionsReq struct {
	token     string
	ChannelID string   `json:"channel_id,omitempty"`
	ThingIDs  []string `json:"thing_ids,omitempty"`
}

func (req connectionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ThingIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	if req.ChannelID == "" {
		return apiutil.ErrMissingID
	}

	for _, thingID := range req.ThingIDs {
		if thingID == "" {
			return apiutil.ErrMissingID
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

type restoreThingReq struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id"`
	Name     string                 `json:"name"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreChannelReq struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type restoreConnectionReq struct {
	ChannelID    string `json:"channel_id"`
	ChannelOwner string `json:"channel_owner"`
	ThingID      string `json:"thing_id"`
	ThingOwner   string `json:"thing_owner"`
}

type restoreGroupReq struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type restoreGroupThingRelationReq struct {
	ThingID   string    `json:"thing_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type restoreGroupChannelRelationReq struct {
	ChannelID string    `json:"channel_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type restoreReq struct {
	token       string
	Things      []restoreThingReq      `json:"things"`
	Channels    []restoreChannelReq    `json:"channels"`
	Connections []restoreConnectionReq `json:"connections"`
	Groups      []restoreGroupReq      `json:"groups"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Groups) == 0 && len(req.Things) == 0 && len(req.Channels) == 0 && len(req.Connections) == 0 {
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
		if len(group.Name) > maxNameSize {
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

type listMembersReq struct {
	token    string
	id       string
	offset   uint64
	limit    uint64
	metadata things.GroupMetadata
}

func (req listMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
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

type getConnByKeyReq struct {
	Key string `json:"key"`
}

func (req getConnByKeyReq) validate() error {
	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type groupRolesReq struct {
	token      string
	groupID    string
	GroupRoles []groupMember `json:"group_roles"`
}

func (req groupRolesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.GroupRoles) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, gp := range req.GroupRoles {
		if gp.Role != things.Viewer && gp.Role != things.Editor {
			return apiutil.ErrInvalidRole
		}

		if gp.ID == "" {
			return apiutil.ErrMissingID
		}
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

type listGroupMembersReq struct {
	token   string
	groupID string
	offset  uint64
	limit   uint64
}

func (req listGroupMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
