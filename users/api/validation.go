package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// ValidatePageMetadata validates the page metadata against the specified allowed orders map.
func ValidatePageMetadata(pm domain.UsersPageMetadata, maxLimitSize, maxEmailSize int, allowedOrders map[string]string) error {
	if len(pm.Email) > maxEmailSize {
		return apiutil.ErrEmailSize
	}

	if pm.Status != "" {
		if pm.Status != domain.AllStatusKey &&
			pm.Status != domain.EnabledStatusKey &&
			pm.Status != domain.DisabledStatusKey {
			return apiutil.ErrInvalidStatus
		}
	}

	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	return common.Validate(maxLimitSize, allowedOrders)
}
