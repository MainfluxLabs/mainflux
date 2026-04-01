package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// Validate validates the page metadata.
func ValidatePageMetadata(pm domain.UsersPageMetadata, maxLimitSize, maxEmailSize int) error {
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

var allowedOrders = map[string]string{
	"id":            "id",
	"email":         "email",
	"invitee_email": "invitee_email",
	"state":         "state",
	"created_at":    "created_at",
}
