// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things/api/http/memberships"
)

var (
	_ apiutil.Response = (*viewThingRes)(nil)
	_ apiutil.Response = (*backupRes)(nil)
)

type viewThingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
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

type backupProfile struct {
	ID       string         `json:"id"`
	GroupID  string         `json:"group_id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type backupGroup struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	OrgID       string         `json:"org_id"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type backupRes struct {
	Things           []viewThingRes                       `json:"things"`
	Profiles         []backupProfile                      `json:"profiles"`
	Groups           []backupGroup                        `json:"groups"`
	GroupMemberships []memberships.ViewGroupMembershipRes `json:"group_memberships"`
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
