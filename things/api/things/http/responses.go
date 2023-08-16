// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux"
)

var (
	_ mainflux.Response = (*viewThingRes)(nil)
	_ mainflux.Response = (*thingsPageRes)(nil)
	_ mainflux.Response = (*viewChannelRes)(nil)
	_ mainflux.Response = (*channelsPageRes)(nil)
	_ mainflux.Response = (*connectThingRes)(nil)
	_ mainflux.Response = (*connectRes)(nil)
	_ mainflux.Response = (*disconnectThingRes)(nil)
	_ mainflux.Response = (*disconnectRes)(nil)
	_ mainflux.Response = (*shareThingRes)(nil)
	_ mainflux.Response = (*backupRes)(nil)
	_ mainflux.Response = (*groupThingsPageRes)(nil)
	_ mainflux.Response = (*groupChannelsPageRes)(nil)
	_ mainflux.Response = (*groupRes)(nil)
	_ mainflux.Response = (*removeRes)(nil)
	_ mainflux.Response = (*assignRes)(nil)
	_ mainflux.Response = (*unassignRes)(nil)
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
	Owner    string                 `json:"-"`
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
	Name     string                 `json:"name,omitempty"`
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

type viewChannelRes struct {
	ID       string                 `json:"id"`
	Owner    string                 `json:"-"`
	Name     string                 `json:"name,omitempty"`
	Things   []viewThingRes         `json:"connected,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (res viewChannelRes) Code() int {
	return http.StatusOK
}

func (res viewChannelRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewChannelRes) Empty() bool {
	return false
}

type channelsPageRes struct {
	pageRes
	Channels []viewChannelRes `json:"channels"`
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

type connectThingRes struct{}

func (res connectThingRes) Code() int {
	return http.StatusOK
}

func (res connectThingRes) Headers() map[string]string {
	return map[string]string{
		"Warning-Deprecated": "This endpoint will be depreciated in v1.0.0. It will be replaced with the bulk endpoint found at /connect.",
	}
}

func (res connectThingRes) Empty() bool {
	return true
}

type connectRes struct{}

func (res connectRes) Code() int {
	return http.StatusOK
}

func (res connectRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectRes) Empty() bool {
	return true
}

type disconnectRes struct{}

func (res disconnectRes) Code() int {
	return http.StatusOK
}

func (res disconnectRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectRes) Empty() bool {
	return true
}

type disconnectThingRes struct{}

func (res disconnectThingRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectThingRes) Empty() bool {
	return true
}

type backupThingRes struct {
	ID       string                 `json:"id"`
	Owner    string                 `json:"owner,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupChannelRes struct {
	ID       string                 `json:"id"`
	Owner    string                 `json:"owner,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type backupConnectionRes struct {
	ChannelID    string `json:"channel_id"`
	ChannelOwner string `json:"channel_owner"`
	ThingID      string `json:"thing_id"`
	ThingOwner   string `json:"thing_owner"`
}

type backupGroupThingRelationRes struct {
	ThingID   string    `json:"thing_id"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type backupRes struct {
	Things         []backupThingRes              `json:"things"`
	Channels       []backupChannelRes            `json:"channels"`
	Connections    []backupConnectionRes         `json:"connections"`
	Groups         []viewGroupRes                `json:"groups"`
	GroupRelations []backupGroupThingRelationRes `json:"group_thing_relations"`
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

type groupThingsPageRes struct {
	pageRes
	Things []thingRes `json:"things"`
}

func (res groupThingsPageRes) Code() int {
	return http.StatusOK
}

func (res groupThingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupThingsPageRes) Empty() bool {
	return false
}

type groupChannelsPageRes struct {
	pageRes
	Channels []channelRes `json:"channels"`
}

func (res groupChannelsPageRes) Code() int {
	return http.StatusOK
}

func (res groupChannelsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupChannelsPageRes) Empty() bool {
	return false
}

type viewGroupRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
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
	id      string
	created bool
}

func (res groupRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res groupRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/groups/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res groupRes) Empty() bool {
	return true
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

type assignRes struct{}

func (res assignRes) Code() int {
	return http.StatusOK
}

func (res assignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignRes) Empty() bool {
	return true
}

type unassignRes struct{}

func (res unassignRes) Code() int {
	return http.StatusNoContent
}

func (res unassignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignRes) Empty() bool {
	return true
}
