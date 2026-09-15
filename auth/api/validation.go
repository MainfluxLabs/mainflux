// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

// ValidatePageMetadata validates the auth page metadata against the specified allowed orders map.
func ValidatePageMetadata(pm auth.PageMetadata, maxLimitSize, maxNameSize int, allowedOrders map[string]string) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, allowedOrders); err != nil {
		return err
	}

	if len(pm.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}
