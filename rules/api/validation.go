// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

// ValidatePageMetadata validates the page metadata against the given set of allowed order fields.
func ValidatePageMetadata(pm rules.PageMetadata, maxLimitSize, maxNameSize int, orderFields map[string]string) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, orderFields); err != nil {
		return err
	}

	if len(pm.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if pm.InputType != "" {
		switch pm.InputType {
		case rules.InputTypeMessage, rules.InputTypeAlarm:
		default:
			return apiutil.ErrInvalidInputType
		}
	}

	return nil
}
