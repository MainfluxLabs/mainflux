package auth

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// RoleRootAdmin and RoleAdmin are aliases for the shared domain types.
const (
	RoleRootAdmin = domain.RoleRootAdmin
	RoleAdmin     = domain.RoleAdmin
)

type Roles interface {
	// AssignRole assigns a role to a user.
	AssignRole(ctx context.Context, id, role string) error

	// RetrieveRole retrieves a role for a user.
	RetrieveRole(ctx context.Context, id string) (string, error)

	// RetrieveRoles retrieves the roles of the users with the provided IDs, keyed by user ID.
	RetrieveRoles(ctx context.Context, ids []string) (map[string]string, error)

	// UnassignRole unassigns the role from a user.
	UnassignRole(ctx context.Context, id string) error
}

type RolesRepository interface {
	// SaveRole saves the user role.
	SaveRole(ctx context.Context, id, role string) error
	// RetrieveRole retrieves the user role.
	RetrieveRole(ctx context.Context, id string) (string, error)
	// RetrieveRoles retrieves the roles of the users with the provided IDs, keyed by user ID.
	RetrieveRoles(ctx context.Context, ids []string) (map[string]string, error)
	// UpdateRole updates the user role.
	UpdateRole(ctx context.Context, id, role string) error
	// RemoveRole removes the user role.
	RemoveRole(ctx context.Context, id string) error
}

func (svc service) AssignRole(ctx context.Context, id, role string) error {
	return svc.roles.SaveRole(ctx, id, role)
}

func (svc service) RetrieveRole(ctx context.Context, id string) (string, error) {
	return svc.roles.RetrieveRole(ctx, id)
}

func (svc service) RetrieveRoles(ctx context.Context, ids []string) (map[string]string, error) {
	return svc.roles.RetrieveRoles(ctx, ids)
}

func (svc service) UnassignRole(ctx context.Context, id string) error {
	return svc.roles.RemoveRole(ctx, id)
}
