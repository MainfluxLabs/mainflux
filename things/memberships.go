package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrGroupMembershipExists indicates that membership already exists.
	ErrGroupMembershipExists = errors.New("group membership already exists")

	// ErrMissingUserMembership indicates that required user membership was not found.
	ErrMissingUserMembership = errors.New("user membership not found")
)

type GroupMembership struct {
	GroupID  string
	MemberID string
	Email    string
	Role     string
}

type GroupMembershipsPage struct {
	apiutil.PageMetadata
	GroupMemberships []GroupMembership
}

// GroupMembershipsRepository specifies an interface for managing group memberships in persistence.
type GroupMembershipsRepository interface {
	// Save persists group memberships.
	Save(ctx context.Context, gms ...GroupMembership) error

	// RetrieveRole retrieves role of a specific group membership.
	RetrieveRole(ctx context.Context, gm GroupMembership) (string, error)

	// RetrieveByGroup retrieves a paginated list of group memberships by group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (GroupMembershipsPage, error)

	// BackupAll retrieves all group memberships. Used for backup.
	BackupAll(ctx context.Context) ([]GroupMembership, error)

	// BackupByGroup retrieves all group memberships by group ID. This is used for backup.
	BackupByGroup(ctx context.Context, groupID string) ([]GroupMembership, error)

	// RetrieveGroupIDsByMember retrieves IDs of groups where the member belongs.
	RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error)

	// Remove removes the provided group memberships.
	Remove(ctx context.Context, groupID string, memberIDs ...string) error

	// Update updates existing group memberships.
	Update(ctx context.Context, gms ...GroupMembership) error
}

// GroupMemberships defines a service for managing group memberships.
type GroupMemberships interface {
	// CreateGroupMemberships adds memberships to a group identified by the provided ID.
	CreateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error

	// ListGroupMemberships retrieves a paginated list of group memberships for the given group.
	ListGroupMemberships(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (GroupMembershipsPage, error)

	// UpdateGroupMemberships updates roles of a specific group membership.
	UpdateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error

	// RemoveGroupMemberships removes memberships from the given group.
	RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error {
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}
		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		if err := ts.groupMemberships.Save(ctx, gm); err != nil {
			return err
		}

		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) ListGroupMemberships(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (GroupMembershipsPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return GroupMembershipsPage{}, err
	}

	groupMemberships, err := ts.groupMemberships.BackupByGroup(ctx, groupID)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	var memberIDs []string
	membershipByMemberID := make(map[string]GroupMembership, len(groupMemberships))
	for _, gm := range groupMemberships {
		memberIDs = append(memberIDs, gm.MemberID)
		membershipByMemberID[gm.MemberID] = gm
	}

	var gms []GroupMembership
	var page *protomfx.UsersRes
	if len(groupMemberships) > 0 {
		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs, Email: pm.Email, Order: pm.Order, Dir: pm.Dir, Limit: pm.Limit, Offset: pm.Offset}
		page, err = ts.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return GroupMembershipsPage{}, err
		}

		for _, u := range page.Users {
			m, ok := membershipByMemberID[u.Id]
			if !ok {
				return GroupMembershipsPage{}, ErrMissingUserMembership
			}

			gm := GroupMembership{
				MemberID: m.MemberID,
				Email:    u.Email,
				Role:     m.Role,
			}

			gms = append(gms, gm)
		}
	}

	gmp := GroupMembershipsPage{
		GroupMemberships: gms,
		PageMetadata: apiutil.PageMetadata{
			Total:  page.Total,
			Offset: page.Offset,
			Limit:  page.Limit,
		},
	}

	return gmp, nil
}

func (ts *thingsService) UpdateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error {
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}

		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		role, err := ts.groupCache.ViewRole(ctx, gm.GroupID, gm.MemberID)
		if err != nil {
			r, err := ts.groupMemberships.RetrieveRole(ctx, gm)
			if err != nil {
				return err
			}
			role = r
		}
		if role == Owner {
			return errors.ErrAuthorization
		}
	}

	if err := ts.groupMemberships.Update(ctx, gms...); err != nil {
		return err
	}

	for _, gm := range gms {
		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Admin,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return err
	}

	for _, mid := range memberIDs {
		role, err := ts.groupCache.ViewRole(ctx, groupID, mid)
		if err != nil {
			r, err := ts.groupMemberships.RetrieveRole(ctx, GroupMembership{GroupID: groupID, MemberID: mid})
			if err != nil {
				return err
			}
			role = r
		}
		if role == Owner {
			return errors.ErrAuthorization
		}
	}

	if err := ts.groupMemberships.Remove(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	for _, mid := range memberIDs {
		if err := ts.groupCache.RemoveGroupMembership(ctx, groupID, mid); err != nil {
			return err
		}
	}

	return nil
}
