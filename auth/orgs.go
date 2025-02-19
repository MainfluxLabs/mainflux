package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	// ErrAssignMember indicates failure to assign member to org.
	ErrAssignMember = errors.New("failed to assign member to org")

	// ErrUnassignMember indicates failure to unassign member from an org.
	ErrUnassignMember = errors.New("failed to unassign member from org")

	// ErrOrgNotEmpty indicates org is not empty, can't be deleted.
	ErrOrgNotEmpty = errors.New("org is not empty")

	// ErrOrgMemberAlreadyAssigned indicates that members is already assigned.
	ErrOrgMemberAlreadyAssigned = errors.New("org member is already assigned")
)

// OrgMetadata defines the Metadata type.
type OrgMetadata map[string]interface{}

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

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Name     string
	Metadata OrgMetadata
}

// OrgsPage contains page related metadata as well as list of orgs that
// belong to this page.
type OrgsPage struct {
	PageMetadata
	Orgs []Org
}

type User struct {
	ID     string
	Email  string
	Status string
}

type Backup struct {
	Orgs       []Org
	OrgMembers []OrgMember
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
	ListOrgs(ctx context.Context, token string, pm PageMetadata) (OrgsPage, error)

	// ListOrgsByMember retrieves all orgs for member that is identified with memberID belongs to.
	ListOrgsByMember(ctx context.Context, token, memberID string, pm PageMetadata) (OrgsPage, error)

	// RemoveOrg removes the org identified with the provided ID.
	RemoveOrg(ctx context.Context, token, id string) error

	// GetOwnerIDByOrgID returns an owner ID for a given org ID.
	GetOwnerIDByOrgID(ctx context.Context, orgID string) (string, error)

	// Backup retrieves all orgs and org members. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds orgs and org members from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error
}

// OrgRepository specifies an org persistence API.
type OrgRepository interface {
	// Save orgs
	Save(ctx context.Context, orgs ...Org) error

	// Update an org
	Update(ctx context.Context, org Org) error

	// Remove an org
	Remove(ctx context.Context, owner, id string) error

	// RetrieveByID retrieves org by its id
	RetrieveByID(ctx context.Context, id string) (Org, error)

	// RetrieveByOwner retrieves orgs by owner.
	RetrieveByOwner(ctx context.Context, ownerID string, pm PageMetadata) (OrgsPage, error)

	// RetrieveAll retrieves all orgs.
	RetrieveAll(ctx context.Context) ([]Org, error)

	// RetrieveByAdmin retrieves all orgs with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (OrgsPage, error)

	// RetrieveByMemberID list of orgs that member belongs to
	RetrieveByMemberID(ctx context.Context, memberID string, pm PageMetadata) (OrgsPage, error)
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

	timestamp := getTimestmap()

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

	om := OrgMember{
		OrgID:     id,
		MemberID:  user.ID,
		Role:      Owner,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	if err := svc.members.Save(ctx, om); err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) ListOrgs(ctx context.Context, token string, pm PageMetadata) (OrgsPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgs.RetrieveByAdmin(ctx, pm)
	}

	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	return svc.orgs.RetrieveByOwner(ctx, user.ID, pm)
}

func (svc service) RemoveOrg(ctx context.Context, token, id string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessOrg(ctx, token, id, Owner); err != nil {
		return err
	}

	return svc.orgs.Remove(ctx, user.ID, id)
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
		UpdatedAt:   getTimestmap(),
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

func (svc service) ListOrgsByMember(ctx context.Context, token string, memberID string, pm PageMetadata) (OrgsPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgs.RetrieveByMemberID(ctx, memberID, pm)
	}

	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	if user.ID != memberID {
		return OrgsPage{}, errors.ErrAuthorization
	}

	return svc.orgs.RetrieveByMemberID(ctx, memberID, pm)
}

func (svc service) GetOwnerIDByOrgID(ctx context.Context, orgID string) (string, error) {
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

	role, err := svc.members.RetrieveRole(ctx, user.ID, orgID)
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
