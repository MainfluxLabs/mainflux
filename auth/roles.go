package auth

import "context"

const (
	// RoleRootAdmin is the super admin role.
	RoleRootAdmin = "root_admin"
	// RoleAdmin is the admin role.
	RoleAdmin = "admin"
)

type RolesService interface {
	// AssignRole assigns the user role.
	AssignRole(ctx context.Context, id, role string) error
}

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
