package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type MockClient struct {
	key   map[string]string
	conns map[string]interface{}
}

func NewClient(key map[string]string, conns map[string]interface{}) auth.Client {
	return MockClient{key: key, conns: conns}
}

func (cli MockClient) Authorize(ctx context.Context, chanID, thingID string) error {
	for k, v := range cli.conns {
		if k == chanID && v == thingID {
			return nil
		}
	}
	return errors.ErrAuthentication
}

func (cli MockClient) Identify(ctx context.Context, thingKey string) (string, error) {
	if id, ok := cli.key[thingKey]; ok {
		return id, nil
	}
	return "", errors.ErrAuthentication
}

type SubjectSet struct {
	Object   string
	Relation string
}

type authServiceMock struct {
	users map[string]string
	authz map[string][]SubjectSet
}

// NewAuth creates mock of auth service.
func NewAuth(users map[string]string, authz map[string][]SubjectSet) mainflux.AuthServiceClient {
	return &authServiceMock{users, authz}
}

func (svc authServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	if req.GetToken() != "token" {
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
}

func (svc authServiceMock) CanAccessGroup(ctx context.Context, in *mainflux.AccessGroupReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) AssignRole(ctx context.Context, req *mainflux.AssignRoleReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}
