// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
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

func (svc authServiceMock) Authorize(_ context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	u, ok := svc.usersByEmail[req.Token]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	switch req.Subject {
	case auth.RootSub:
		if !contains(svc.roles[auth.RootSub], u.ID) {
			return &empty.Empty{}, errors.ErrAuthorization
		}
	case auth.OrgSub:
		if err := svc.canAccessOrg(u.ID, req.Action); err != nil {
			return &empty.Empty{}, err
		}
	default:
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
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

func (svc authServiceMock) GetOwnerIDByOrgID(_ context.Context, req *protomfx.OrgID, _ ...grpc.CallOption) (*protomfx.OwnerID, error) {
	for id, org := range svc.orgs {
		if id == req.Value {
			return &protomfx.OwnerID{Value: org.OwnerID}, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (svc authServiceMock) AssignRole(_ context.Context, _ *protomfx.AssignRoleReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(_ context.Context, _ *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	panic("not implemented")
}
