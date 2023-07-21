package auth

import "context"

const (
	// RoleRootAdmin is the super admin role.
	RoleRootAdmin = "root"
	// RoleAdmin is the admin role.
	RoleAdmin = "admin"
)

type RolesRepository interface {
	// SaveRole saves the user role.
	SaveRole(ctx context.Context, id, role string) error
	// RetrieveRole retrieves the user role.
	RetrieveRole(ctx context.Context, id string) (string, error)
	// UpdateRole updates the user role.
	UpdateRole(ctx context.Context, id, role string) error
	// RemoveRole removes the user role.
	RemoveRole(ctx context.Context, id string) error
}
