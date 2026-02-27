// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.AuthServiceClient = (*authServiceMock)(nil)

type authServiceMock struct {
	roles        map[string][]string
	usersByEmail map[string]users.User
	orgs         map[string]auth.Org
}

// NewAuthService creates mock of users service.
func NewAuthService(adminID string, userList []users.User, orgList []auth.Org) protomfx.AuthServiceClient {
	usersByEmail := make(map[string]users.User)
	roles := map[string][]string{auth.RootSub: {adminID}}
	orgs := make(map[string]auth.Org)

	for _, user := range userList {
		usersByEmail[user.Email] = user
		roles[user.Role] = append(roles[user.Role], user.ID)
	}

	for _, o := range orgList {
		orgs[o.ID] = o
	}

	return &authServiceMock{
		roles:        roles,
		usersByEmail: usersByEmail,
		orgs:         orgs,
	}
}

func (svc authServiceMock) Identify(_ context.Context, in *protomfx.Token, _ ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	if u, ok := svc.usersByEmail[in.Value]; ok {
		return &protomfx.UserIdentity{Id: u.ID, Email: u.Email}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(_ context.Context, in *protomfx.IssueReq, _ ...grpc.CallOption) (*protomfx.Token, error) {
	if u, ok := svc.usersByEmail[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &protomfx.Token{Value: u.Email}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(_ context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	u, ok := svc.usersByEmail[req.Token]
	if !ok {
		return &emptypb.Empty{}, errors.ErrAuthentication
	}

	switch req.Subject {
	case auth.RootSub:
		if !contains(svc.roles[auth.RootSub], u.ID) {
			return &emptypb.Empty{}, errors.ErrAuthorization
		}
	case auth.OrgSub:
		if err := svc.canAccessOrg(u.ID, req.Action); err != nil {
			return &emptypb.Empty{}, err
		}
	default:
		return &emptypb.Empty{}, errors.ErrAuthorization
	}

	return &emptypb.Empty{}, nil
}

func contains(ids []string, id string) bool {
	for _, existingID := range ids {
		if existingID == id {
			return true
		}
	}
	return false
}

func (svc authServiceMock) canAccessOrg(userID, action string) error {
	isRoot := contains(svc.roles[auth.RootSub], userID)
	isOwner := isRoot || contains(svc.roles[auth.Owner], userID)
	isEditor := isOwner || contains(svc.roles[auth.Editor], userID)
	isViewer := isEditor || contains(svc.roles[auth.Viewer], userID)

	switch action {
	case auth.RootSub:
		if !isRoot {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Owner:
		if !isOwner {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Editor:
		if !isEditor {
			return errors.ErrAuthorization
		}
		return nil
	case auth.Viewer:
		if !isViewer {
			return errors.ErrAuthorization
		}
		return nil
	default:
		return errors.ErrAuthorization
	}
}

func (svc authServiceMock) GetOwnerIDByOrg(_ context.Context, req *protomfx.OrgID, _ ...grpc.CallOption) (*protomfx.OwnerID, error) {
	for id, org := range svc.orgs {
		if id == req.Value {
			return &protomfx.OwnerID{Value: org.OwnerID}, nil
		}
	}
	return nil, dbutil.ErrNotFound
}

func (svc authServiceMock) AssignRole(context.Context, *protomfx.AssignRoleReq, ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(context.Context, *protomfx.RetrieveRoleReq, ...grpc.CallOption) (*protomfx.RetrieveRoleRes, error) {
	panic("not implemented")
}

func (svc authServiceMock) CreateDormantOrgInvite(context.Context, *protomfx.CreateDormantOrgInviteReq, ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}

func (svc authServiceMock) ActivateOrgInvite(context.Context, *protomfx.ActivateOrgInviteReq, ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}

func (svc authServiceMock) GetDormantInviteByPlatformInvite(context.Context, *protomfx.GetDormantInviteByPlatformInviteReq, ...grpc.CallOption) (*protomfx.OrgInvite, error) {
	return nil, status.Error(codes.NotFound, dbutil.ErrNotFound.Error())
}

func (svc authServiceMock) ViewOrg(_ context.Context, req *protomfx.ViewOrgReq, _ ...grpc.CallOption) (r *protomfx.Org, err error) {
	org, ok := svc.orgs[req.GetOrgID()]
	if !ok {
		return nil, dbutil.ErrNotFound
	}

	return &protomfx.Org{
		Id:      org.ID,
		OwnerID: org.OwnerID,
		Name:    org.Name,
	}, nil
}
