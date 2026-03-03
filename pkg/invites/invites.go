// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/roles"
)

type OrgInvite struct {
	ID           string
	InviteeID    string
	InviteeEmail string
	InviterID    string
	InviterEmail string
	OrgID        string
	OrgName      string
	InviteeRole  string
	GroupInvites []GroupInvite
	CreatedAt    time.Time
	ExpiresAt    time.Time
	State        string
}

type GroupInvite struct {
	GroupID    string `json:"group_id"`
	MemberRole string `json:"member_role"`
}

func ValidateInviteeRole(role string) error {
	if role != roles.OrgAdmin && role != roles.OrgEditor && role != roles.OrgViewer {
		return apiutil.ErrInvalidRole
	}

	return nil
}
