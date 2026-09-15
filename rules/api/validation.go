// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

// ValidatePageMetadata validates the rules page metadata.
func ValidatePageMetadata(pm rules.PageMetadata, maxLimitSize, maxNameSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, rules.RuleOrderFields); err != nil {
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
