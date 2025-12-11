package things

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// ErrGroupMembershipExists indicates that membership already exists.
var ErrGroupMembershipExists = errors.New("group membership already exists")

type GroupMembership struct {
	GroupID  string
	MemberID string
	Email    string
	Role     string
}

type GroupMembershipsPage struct {
	Total            uint64
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

	// CreateGroupMembershipsInternal saves group memberships without requiring authentication.
	CreateGroupMembershipsInternal(ctx context.Context, gms ...GroupMembership) error

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

		group, err := ts.groups.RetrieveByID(ctx, gm.GroupID)
		if err != nil {
			return err
		}

		if err := ts.groupMemberships.Save(ctx, gm); err != nil {
			return err
		}

		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}

		org, err := ts.auth.ViewOrg(ctx, &protomfx.ViewOrgReq{
			Token: token,
			OrgID: group.OrgID,
		})

		if err != nil {
			fmt.Println(err)
			continue
		}

		users, err := ts.users.GetUsersByIDs(ctx, &protomfx.UsersByIDsReq{
			Ids: []string{gm.MemberID},
		})

		if err != nil {
			fmt.Println(err)
			continue
		}

		recipientEmail := users.GetUsers()[0].Email

		// Send e-mail notification
		go func() {
			ts.email.SendGroupMembershipNotification([]string{recipientEmail}, org.Name, group.Name, gm.Role)
		}()
	}

	return nil
}

func (ts *thingsService) CreateGroupMembershipsInternal(ctx context.Context, gms ...GroupMembership) error {
	for _, gm := range gms {
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

	memberships, err := ts.groupMemberships.BackupByGroup(ctx, groupID)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	if len(memberships) == 0 {
		return GroupMembershipsPage{
			GroupMemberships: []GroupMembership{},
			Total:            0,
		}, nil
	}

	memberIDs := make([]string, 0, len(memberships))
	membershipByMemberID := make(map[string]GroupMembership, len(memberships))
	for _, m := range memberships {
		memberIDs = append(memberIDs, m.MemberID)
		membershipByMemberID[m.MemberID] = m
	}

	userReq := &protomfx.UsersByIDsReq{
		Ids: memberIDs,
		PageMetadata: &protomfx.PageMetadata{
			Email:  pm.Email,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Limit:  pm.Limit,
			Offset: pm.Offset,
		},
	}

	res, err := ts.users.GetUsersByIDs(ctx, userReq)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	var gms []GroupMembership
	for _, u := range res.Users {
		if m, ok := membershipByMemberID[u.Id]; ok {
			m.Email = u.Email
			gms = append(gms, m)
		}
	}

	return GroupMembershipsPage{
		GroupMemberships: gms,
		Total:            res.PageMetadata.Total,
	}, nil
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
