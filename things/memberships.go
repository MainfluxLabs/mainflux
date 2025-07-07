package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
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

	// RetrieveAll retrieves all group memberships. Used for backup.
	RetrieveAll(ctx context.Context) ([]GroupMembership, error)

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

	// UpdateGroupMemberships updates roles of members in the specified group.
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

	gmp, err := ts.groupMemberships.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	var memberIDs []string
	roles := make(map[string]string)
	for _, gm := range gmp.GroupMemberships {
		memberIDs = append(memberIDs, gm.MemberID)
		roles[gm.MemberID] = gm.Role
	}

	var gms []GroupMembership
	if len(gmp.GroupMemberships) > 0 {
		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs, Email: pm.Email, Order: pm.Order, Dir: pm.Dir}
		up, err := ts.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return GroupMembershipsPage{}, err
		}

		for _, user := range up.Users {
			role, ok := roles[user.Id]
			if !ok {
				continue
			}

			gm := GroupMembership{
				MemberID: user.Id,
				Email:    user.Email,
				Role:     role,
			}

			gms = append(gms, gm)
		}
	}

	page := GroupMembershipsPage{
		GroupMemberships: gms,
		PageMetadata: apiutil.PageMetadata{
			Total:  gmp.Total,
			Offset: gmp.Offset,
			Limit:  gmp.Limit,
		},
	}

	return page, nil
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
