// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*viewThingRes)(nil)
	_ apiutil.Response = (*thingsPageRes)(nil)
	_ apiutil.Response = (*channelsPageRes)(nil)
	_ apiutil.Response = (*connectionsRes)(nil)
	_ apiutil.Response = (*shareThingRes)(nil)
	_ apiutil.Response = (*backupRes)(nil)
	_ apiutil.Response = (*ThingsPageRes)(nil)
	_ apiutil.Response = (*groupsRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
	_ apiutil.Response = (*listGroupRolesRes)(nil)
	_ apiutil.Response = (*updateGroupRolesRes)(nil)
	_ apiutil.Response = (*createGroupRolesRes)(nil)
)

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}

type shareThingRes struct{}

func (res shareThingRes) Code() int {
	return http.StatusOK
}

func (res shareThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res shareThingRes) Empty() bool {
	return false
}

type thingRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	created  bool
}

type thingsRes struct {
	Things  []thingRes `json:"things"`
	created bool
}

func (res thingsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res thingsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsRes) Empty() bool {
	return false
}

type viewThingRes struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"-"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (res viewThingRes) Code() int {
	return http.StatusOK
}

func (res viewThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewThingRes) Empty() bool {
	return false
}

type thingsPageRes struct {
	pageRes
	Things []viewThingRes `json:"things"`
}

func (res thingsPageRes) Code() int {
	return http.StatusOK
}

func (res thingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsPageRes) Empty() bool {
	return false
}

type channelRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	created  bool
}

type channelsRes struct {
	Channels []channelRes `json:"channels"`
	created  bool
}

func (res channelsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res channelsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res channelsRes) Empty() bool {
	return false
}

type channelsPageRes struct {
	pageRes
	Channels []channelRes `json:"channels"`
}

func (res channelsPageRes) Code() int {
	return http.StatusOK
}

func (res channelsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res channelsPageRes) Empty() bool {
	return false
}

type connectionsRes struct{}

func (res connectionsRes) Code() int {
	return http.StatusOK
}

func (res connectionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectionsRes) Empty() bool {
	return true
}

type backupThingRes struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupChannelRes struct {
	ID       string                 `json:"id"`
	OwnerID  string                 `json:"owner_id,omitempty"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Profile  map[string]interface{} `json:"profile,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupConnectionRes struct {
	ChannelID string `json:"channel_id"`
	ThingID   string `json:"thing_id"`
}

type backupGroupThingRelationRes struct {
	ThingID   string    `json:"thing_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type backupGroupChannelRelationRes struct {
	ChannelID string    `json:"channel_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type backupRes struct {
	Things                []backupThingRes                `json:"things"`
	Channels              []backupChannelRes              `json:"channels"`
	Connections           []backupConnectionRes           `json:"connections"`
	Groups                []viewGroupRes                  `json:"groups"`
	GroupThingRelations   []backupGroupThingRelationRes   `json:"group_thing_relations"`
	GroupChannelRelations []backupGroupChannelRelationRes `json:"group_channel_relations"`
}

func (res backupRes) Code() int {
	return http.StatusOK
}

func (res backupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res backupRes) Empty() bool {
	return false
}

type restoreRes struct{}

func (res restoreRes) Code() int {
	return http.StatusCreated
}

func (res restoreRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreRes) Empty() bool {
	return true
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order"`
	Dir    string `json:"direction"`
	Name   string `json:"name"`
}

type ThingsPageRes struct {
	pageRes
	Things []thingRes `json:"things"`
}

func (res ThingsPageRes) Code() int {
	return http.StatusOK
}

func (res ThingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res ThingsPageRes) Empty() bool {
	return false
}

type viewGroupRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
	OrgID       string                 `json:"org_id"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (res viewGroupRes) Code() int {
	return http.StatusOK
}

func (res viewGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewGroupRes) Empty() bool {
	return false
}

type groupRes struct {
	ID          string                 `json:"id"`
	OrgID       string                 `json:"org_id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	created     bool
}

type groupsRes struct {
	Groups  []groupRes `json:"groups"`
	created bool
}

func (res groupsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res groupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupsRes) Empty() bool {
	return false
}

type groupPageRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

func (res groupPageRes) Code() int {
	return http.StatusOK
}

func (res groupPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupPageRes) Empty() bool {
	return false
}

type identityRes struct {
	ID string `json:"id"`
}

func (res identityRes) Code() int {
	return http.StatusOK
}

func (res identityRes) Headers() map[string]string {
	return map[string]string{}
}

func (res identityRes) Empty() bool {
	return false
}

type connByKeyRes struct {
	ChannelID string `json:"channel_id"`
	ThingID   string `json:"thing_id"`
}

func (res connByKeyRes) Code() int {
	return http.StatusOK
}

func (res connByKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connByKeyRes) Empty() bool {
	return false
}

type groupMember struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type createGroupRolesRes struct{}

func (res createGroupRolesRes) Code() int {
	return http.StatusCreated
}

func (res createGroupRolesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createGroupRolesRes) Empty() bool {
	return true
}

type updateGroupRolesRes struct{}

func (res updateGroupRolesRes) Code() int {
	return http.StatusOK
}

func (res updateGroupRolesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateGroupRolesRes) Empty() bool {
	return true
}

type listGroupRolesRes struct {
	pageRes
	GroupRoles []groupMember `json:"group_roles"`
}

func (res listGroupRolesRes) Code() int {
	return http.StatusOK
}

func (res listGroupRolesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listGroupRolesRes) Empty() bool {
	return false
}
