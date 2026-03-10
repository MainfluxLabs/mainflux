// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// Thing represents a Mainflux thing. Each thing is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Thing struct {
	ID          string          `json:"id,omitempty"`
	GroupID     string          `json:"group_id,omitempty"`
	ProfileID   string          `json:"profile_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Key         string          `json:"key,omitempty"`
	ExternalKey string          `json:"external_key,omitempty"`
	Metadata    domain.Metadata `json:"metadata,omitempty"`
}

// ThingsPage contains page related metadata as well as list of things that
// belong to this page.
type ThingsPage struct {
	Total  uint64  `json:"total"`
	Things []Thing `json:"things"`
}

// Thing key type constants.
const (
	KeyTypeInternal = "internal"
	KeyTypeExternal = "external"
)

// ThingKey represents a Thing authentication key and its type.
type ThingKey struct {
	Value string `json:"key"`
	Type  string `json:"type"`
}

// Validate returns an error if the thing key is invalid.
func (tk ThingKey) Validate() error {
	if tk.Type != KeyTypeExternal && tk.Type != KeyTypeInternal {
		return apiutil.ErrInvalidThingKeyType
	}
	if tk.Value == "" {
		return apiutil.ErrBearerKey
	}
	return nil
}

// ExtractThingKeyFromHTTPHeader returns the thing key and its type from the request's HTTP 'Authorization' header.
// If the provided key type is invalid, an empty ThingKey is returned.
func ExtractThingKeyFromHTTPHeader(r *http.Request) ThingKey {
	header := r.Header.Get("Authorization")

	switch {
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixInternal):
		return ThingKey{
			Type:  KeyTypeInternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixInternal),
		}
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixExternal):
		return ThingKey{
			Type:  KeyTypeExternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixExternal),
		}
	}

	return ThingKey{}
}

// Profile represents a communication group (things that can exchange messages).
type Profile struct {
	ID       string          `json:"id,omitempty"`
	GroupID  string          `json:"group_id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Config   map[string]any  `json:"config,omitempty"`
	Metadata domain.Metadata `json:"metadata,omitempty"`
}

// Group represents group information.
type Group struct {
	ID          string          `json:"id,omitempty"`
	OrgID       string          `json:"org_id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Metadata    domain.Metadata `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GroupPage contains page metadata and list of groups.
type GroupPage struct {
	Total  uint64  `json:"total"`
	Groups []Group `json:"groups"`
}

// ProfilesPage contains page metadata and list of profiles.
type ProfilesPage struct {
	Total    uint64    `json:"total"`
	Profiles []Profile `json:"profiles"`
}

// UserAccessReq represents a user access request.
type UserAccessReq struct {
	Token  string
	ID     string
	Action string
}

// ThingAccessReq represents a thing access request.
type ThingAccessReq struct {
	ThingKey
	ID string
}

// OrgAccessReq represents an org access request.
type OrgAccessReq struct {
	OrgID string
	Token string
}

// PubConfigInfo represents publisher config from GetPubConfigByKey.
type PubConfigInfo struct {
	PublisherID   string
	ProfileConfig map[string]any
}

// Config represents profile configuration.
type Config struct {
	ContentType string      `json:"content_type"`
	Transformer Transformer `json:"transformer"`
}

// Transformer represents message transformation config.
type Transformer struct {
	DataFilters  []string `json:"data_filters"`
	DataField    string   `json:"data_field"`
	TimeField    string   `json:"time_field"`
	TimeFormat   string   `json:"time_format"`
	TimeLocation string   `json:"time_location"`
}

// GroupMembership represents a group membership.
type GroupMembership struct {
	GroupID  string `json:"group_id,omitempty"`
	MemberID string `json:"member_id,omitempty"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}

// GroupMembershipsPage contains page metadata and list of group memberships.
type GroupMembershipsPage struct {
	Total            uint64            `json:"total"`
	GroupMemberships []GroupMembership `json:"group_memberships"`
}

// Role constants for group membership.
const (
	Viewer = "viewer"
	Editor = "editor"
	Admin  = "admin"
	Owner  = "owner"
)
