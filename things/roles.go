package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type GroupMember struct {
	GroupID  string
	MemberID string
	Email    string
	Role     string
}

type GroupRoles struct {
	MemberID string
	Role     string
}
type GroupMembersPage struct {
	PageMetadata
	GroupMembers []GroupMember
}

type RolesRepository interface {
	// SaveRolesByGroup saves group roles by group ID.
	SaveRolesByGroup(ctx context.Context, groupID string, gps ...GroupRoles) error

	// RetrieveRole retrieves group role by group ID.
	RetrieveRole(ctc context.Context, gp GroupMember) (string, error)

	// RetrieveRolesByGroup retrieves page of group roles by groupID.
	RetrieveRolesByGroup(ctx context.Context, groupID string, pm PageMetadata) (GroupMembersPage, error)

	// RetrieveAllRolesByGroup retrieves all group roles by group ID. This is used for backup.
	RetrieveAllRolesByGroup(ctx context.Context) ([]GroupMember, error)

	// RetrieveGroupIDsByMember retrieves the IDs of the groups to which the member belongs
	RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error)

	// RemoveRolesByGroup removes group roles by group ID.
	RemoveRolesByGroup(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdateRolesByGroup updates group roles by group ID.
	UpdateRolesByGroup(ctx context.Context, groupID string, gps ...GroupRoles) error
}

type Roles interface {
	// CreateRolesByGroup creates roles of the group identified by the provided ID.
	CreateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error

	// ListRolesByGroup retrieves a page of roles for a group that is identified by the provided ID.
	ListRolesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupMembersPage, error)

	// UpdateRolesByGroup updates roles of the group identified by the provided ID.
	UpdateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error

	// RemoveRolesByGroup removes roles of the group identified by the provided ID.
	RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Admin,
	}

	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	if err := ts.roles.SaveRolesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) ListRolesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupMembersPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Viewer,
	}

	if err := ts.Authorize(ctx, ar); err != nil {
		return GroupMembersPage{}, err
	}

	gpp, err := ts.roles.RetrieveRolesByGroup(ctx, groupID, pm)
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
		PageMetadata: PageMetadata{
			Total:  gpp.Total,
			Offset: gpp.Offset,
			Limit:  gpp.Limit,
		},
	}

	return page, nil
}

func (ts *thingsService) UpdateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Admin,
	}

	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	for _, gp := range gps {
		if gp.MemberID == group.OrgID {
			return errors.ErrAuthorization
		}
	}

	if err := ts.roles.UpdateRolesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Admin,
	}

	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	for _, m := range memberIDs {
		if m == group.OwnerID {
			return errors.ErrAuthorization
		}
	}

	if err := ts.roles.RemoveRolesByGroup(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	return nil
}
