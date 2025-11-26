package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.AuthServiceClient = (*authServiceMock)(nil)

type SubjectSet struct {
	Object   string
	Relation string
}

type authServiceMock struct {
	users map[string]string
	authz map[string][]SubjectSet
}

// NewAuth creates mock of auth service.
func NewAuth(users map[string]string, authz map[string][]SubjectSet) protomfx.AuthServiceClient {
	return &authServiceMock{users, authz}
}

func (svc authServiceMock) Identify(_ context.Context, in *protomfx.Token, _ ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &protomfx.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(_ context.Context, in *protomfx.IssueReq, _ ...grpc.CallOption) (*protomfx.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &protomfx.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(_ context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	if req.GetToken() != "token" {
		return &emptypb.Empty{}, errors.ErrAuthorization
	}

	return &emptypb.Empty{}, nil
}

func (svc authServiceMock) GetOwnerIDByOrgID(_ context.Context, _ *protomfx.OrgID, _ ...grpc.CallOption) (*protomfx.OwnerID, error) {
	panic("not implemented")
}

func (svc authServiceMock) AssignRole(_ context.Context, _ *protomfx.AssignRoleReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(_ context.Context, _ *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) CreateDormantOrgInvite(ctx context.Context, req *protomfx.CreateDormantOrgInviteReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) ActivateOrgInvite(ctx context.Context, req *protomfx.ActivateOrgInviteReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	panic("not implemented")
}
