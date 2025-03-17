package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type GroupMember struct {
	GroupID  string
	MemberID string
	Email    string
	Role     string
}

type GroupMembersPage struct {
	apiutil.PageMetadata
	GroupMembers []GroupMember
}

type GroupMembersRepository interface {
	// Save saves group members.
	Save(ctx context.Context, gms ...GroupMember) error

	// RetrieveRole retrieves group role by group ID and member ID.
	RetrieveRole(ctc context.Context, gp GroupMember) (string, error)

	// RetrieveByGroup retrieves page of group members by group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (GroupMembersPage, error)

	// RetrieveAll retrieves all group members. This is used for backup.
	RetrieveAll(ctx context.Context) ([]GroupMember, error)

	// RetrieveGroupIDsByMember retrieves the IDs of the groups to which the member belongs
	RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error)

	// Remove removes group members.
	Remove(ctx context.Context, groupID string, memberIDs ...string) error

	// Update updates group members.
	Update(ctx context.Context, gms ...GroupMember) error
}

type GroupMembers interface {
	// CreateGroupMembers creates members of the group identified by the provided ID.
	CreateGroupMembers(ctx context.Context, token string, gms ...GroupMember) error

	// ListGroupMembers retrieves a page of members for a group that is identified by the provided ID.
	ListGroupMembers(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (GroupMembersPage, error)

	// UpdateGroupMembers updates members of the group identified by the provided ID.
	UpdateGroupMembers(ctx context.Context, token string, gms ...GroupMember) error

	// RemoveGroupMembers removes members of the group identified by the provided ID.
	RemoveGroupMembers(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreateGroupMembers(ctx context.Context, token string, gms ...GroupMember) error {
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}
		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		if err := ts.groupMembers.Save(ctx, gm); err != nil {
			return err
		}

		if err := ts.groupCache.SaveGroupMember(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) ListGroupMembers(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (GroupMembersPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return GroupMembersPage{}, err
	}

	gpp, err := ts.groupMembers.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return GroupMembersPage{}, err
	}

	var memberIDs []string
	for _, gp := range gpp.GroupMembers {
		memberIDs = append(memberIDs, gp.MemberID)
	}

	var gms []GroupMember
	if len(gpp.GroupMembers) > 0 {
		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs}
		up, err := ts.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return GroupMembersPage{}, err
		}

		emails := make(map[string]string)
		for _, user := range up.Users {
			emails[user.Id] = user.GetEmail()
		}

		for _, gp := range gpp.GroupMembers {
			email, ok := emails[gp.MemberID]
			if !ok {
				return GroupMembersPage{}, err
			}

			gm := GroupMember{
				MemberID: gp.MemberID,
				Email:    email,
				Role:     gp.Role,
			}

			gms = append(gms, gm)
		}
	}

	page := GroupMembersPage{
		GroupMembers: gms,
		PageMetadata: apiutil.PageMetadata{
			Total:  gpp.Total,
			Offset: gpp.Offset,
			Limit:  gpp.Limit,
		},
	}

	return page, nil
}

func (ts *thingsService) UpdateGroupMembers(ctx context.Context, token string, gms ...GroupMember) error {
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}

		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		rm, err := ts.groupCache.ViewRole(ctx, gm.GroupID, gm.MemberID)
		if err != nil {
			r, err := ts.groupMembers.RetrieveRole(ctx, gm)
			if err != nil {
				return err
			}
			rm = r
		}
		if rm == Owner {
			return errors.ErrAuthorization
		}
	}

	if err := ts.groupMembers.Update(ctx, gms...); err != nil {
		return err
	}

	for _, gm := range gms {
		if err := ts.groupCache.SaveGroupMember(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) RemoveGroupMembers(ctx context.Context, token, groupID string, memberIDs ...string) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Admin,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return err
	}

	for _, mid := range memberIDs {
		rm, err := ts.groupCache.ViewRole(ctx, groupID, mid)
		if err != nil {
			r, err := ts.groupMembers.RetrieveRole(ctx, GroupMember{GroupID: groupID, MemberID: mid})
			if err != nil {
				return err
			}
			rm = r
		}
		if rm == Owner {
			return errors.ErrAuthorization
		}
	}
	if err := ts.groupMembers.Remove(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	for _, mID := range memberIDs {
		if err := ts.groupCache.RemoveGroupMember(ctx, groupID, mID); err != nil {
			return err
		}
	}

	return nil
}
