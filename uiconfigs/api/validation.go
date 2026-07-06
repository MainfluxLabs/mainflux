// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var allowedOrders = map[string]string{
	"id":   "id",
	"name": "name",
}

// ValidatePageMetadata validates the uiconfigs page metadata.
func ValidatePageMetadata(pm apiutil.PageMetadata, maxLimitSize int) error {
	return pm.Validate(maxLimitSize, allowedOrders)
}
