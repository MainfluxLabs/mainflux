// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

// ValidateInviteeRole returns an error if the role is not a valid invitee role
// (Admin, Editor, or Viewer). Owner cannot be assigned via invite.
func ValidateInviteeRole(role string) error {
	if role != Admin && role != Editor && role != Viewer {
		return apiutil.ErrInvalidRole
	}
	return nil
}
