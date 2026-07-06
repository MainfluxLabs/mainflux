// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var allowedOrders = map[string]string{
	"id":         "id",
	"name":       "name",
	"created_at": "created_at",
	"updated_at": "updated_at",
	"invitee_id": "invitee_id",
	"inviter_id": "inviter_id",
	"org_id":     "org_id",
	"state":      "state",
	"issued_at":  "issued_at",
}

// ValidatePageMetadata validates the auth page metadata.
func ValidatePageMetadata(pm auth.PageMetadata, maxLimitSize, maxNameSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, allowedOrders); err != nil {
		return err
	}

	if len(pm.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}
