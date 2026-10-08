// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pyscripts"
)

// ValidateScriptPageMetadata validates the scripts page metadata.
func ValidateScriptPageMetadata(pm pyscripts.PageMetadata, maxLimitSize, maxNameSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, pyscripts.ScriptOrderFields); err != nil {
		return err
	}

	if len(pm.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

// ValidateRunPageMetadata validates the script runs page metadata.
func ValidateRunPageMetadata(pm pyscripts.PageMetadata, maxLimitSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, pyscripts.ScriptRunOrderFields); err != nil {
		return err
	}

	if pm.Status != "" {
		switch pm.Status {
		case pyscripts.ScriptRunStatusSuccess, pyscripts.ScriptRunStatusFail:
		default:
			return apiutil.ErrInvalidQueryParams
		}
	}

	return nil
}
