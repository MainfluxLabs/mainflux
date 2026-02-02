package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	// ErrOrgNotEmpty indicates org is not empty, can't be deleted.
	ErrOrgNotEmpty = errors.New("org is not empty")
)

// OrgMetadata defines the Metadata type.
type OrgMetadata map[string]any

// Org represents the org information.
type Org struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
	Metadata    OrgMetadata
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OrgsPage contains page related metadata as well as list of orgs that
// belong to this page.
type OrgsPage struct {
	Total uint64
	Orgs  []Org
}

type User struct {
	ID     string
	Email  string
	Status string
}

type Backup struct {
	Orgs           []Org
	OrgMemberships []OrgMembership
}

// Orgs specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Orgs interface {
	// CreateOrg creates new org.
	CreateOrg(ctx context.Context, token string, org Org) (Org, error)

	// UpdateOrg updates the org identified by the provided ID.
	UpdateOrg(ctx context.Context, token string, org Org) (Org, error)

	// ViewOrg retrieves data about the org identified by ID.
	ViewOrg(ctx context.Context, token, id string) (Org, error)

	// ListOrgs retrieves orgs.
	ListOrgs(ctx context.Context, token string, pm apiutil.PageMetadata) (OrgsPage, error)

	// RemoveOrgs removes the orgs identified with the provided IDs.
	RemoveOrgs(ctx context.Context, token string, ids ...string) error

	// GetOwnerIDByOrg returns an owner ID for a given org ID.
	GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error)

	// Backup retrieves all orgs and org memberships. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds orgs and org memberships from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error
}

// OrgRepository specifies an org persistence API.
type OrgRepository interface {
	// Save orgs
	Save(ctx context.Context, orgs ...Org) error

	// Update an org
	Update(ctx context.Context, org Org) error

	// Remove orgs
	Remove(ctx context.Context, ownerID string, orgIDs ...string) error

	// RetrieveByID retrieves org by its id
	RetrieveByID(ctx context.Context, id string) (Org, error)

	// BackupAll retrieves all orgs.
	BackupAll(ctx context.Context) ([]Org, error)

	// RetrieveAll retrieves all orgs with pagination.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (OrgsPage, error)

	// RetrieveByMember list of orgs that member belongs to
	RetrieveByMember(ctx context.Context, memberID string, pm apiutil.PageMetadata) (OrgsPage, error)
}

func (svc service) CreateOrg(ctx context.Context, token string, o Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return Org{}, err
	}

	timestamp := getTimestamp()

	org := Org{
		ID:          id,
		OwnerID:     user.ID,
		Name:        o.Name,
		Description: o.Description,
		Metadata:    o.Metadata,
		UpdatedAt:   timestamp,
		CreatedAt:   timestamp,
	}

	if err := svc.orgs.Save(ctx, org); err != nil {
		return Org{}, err
	}

	om := OrgMembership{
		OrgID:     id,
		MemberID:  user.ID,
		Role:      Owner,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	if err := svc.memberships.Save(ctx, om); err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) ListOrgs(ctx context.Context, token string, pm apiutil.PageMetadata) (OrgsPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgs.RetrieveAll(ctx, pm)
	}

	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	return svc.orgs.RetrieveByMember(ctx, user.ID, pm)
}

func (svc service) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := svc.canAccessOrg(ctx, token, id, Owner); err != nil {
			return err
		}
	}

	return svc.orgs.Remove(ctx, user.ID, ids...)
}

func (svc service) UpdateOrg(ctx context.Context, token string, o Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	if err := svc.canAccessOrg(ctx, token, o.ID, Admin); err != nil {
		return Org{}, err
	}

	org := Org{
		ID:          o.ID,
		OwnerID:     user.ID,
		Name:        o.Name,
		Description: o.Description,
		Metadata:    o.Metadata,
		UpdatedAt:   getTimestamp(),
	}

	if err := svc.orgs.Update(ctx, org); err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) ViewOrg(ctx context.Context, token, id string) (Org, error) {
	if err := svc.canAccessOrg(ctx, token, id, Viewer); err != nil {
		return Org{}, err
	}

	org, err := svc.orgs.RetrieveByID(ctx, id)
	if err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error) {
	org, err := svc.orgs.RetrieveByID(ctx, orgID)
	if err != nil {
		return "", err
	}

	return org.OwnerID, nil
}

func (svc service) canAccessOrg(ctx context.Context, token, orgID, action string) error {
	if err := svc.isAdmin(ctx, token); err == nil {
		return nil
	}

	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	role, err := svc.memberships.RetrieveRole(ctx, user.ID, orgID)
	if err != nil {
		return err
	}

	switch role {
	case Owner:
		return nil
	case Admin:
		if action != Owner {
			return nil
		}
	case Editor:
		if action == Viewer || action == Editor {
			return nil
		}
	case Viewer:
		if action == Viewer {
			return nil
		}
	}

	return errors.ErrAuthorization
}
