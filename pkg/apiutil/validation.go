package apiutil

import "github.com/MainfluxLabs/mainflux/pkg/domain"

// ValidateOrgInviteeRole returns an error if the role is not a valid invitee role
// (OrgAdmin, OrgEditor, or OrgViewer). OrgOwner cannot be assigned via invite.
func ValidateOrgInviteeRole(role string) error {
	if role != domain.OrgAdmin && role != domain.OrgEditor && role != domain.OrgViewer {
		return ErrInvalidRole
	}
	return nil
}

// ValidateThingKey returns an API validation error if the thing key is invalid.
func ValidateThingKey(key domain.ThingKey) error {
	if key.Type != domain.KeyTypeExternal && key.Type != domain.KeyTypeInternal {
		return ErrInvalidThingKeyType
	}
	if key.Value == "" {
		return ErrBearerKey
	}
	return nil
}
