package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ protomfx.AuthServiceClient = (*authServiceMock)(nil)

type MockClient struct {
	key   map[string]string
	conns map[string]string
}

func NewClient(key map[string]string, conns map[string]string) auth.Client {
	return MockClient{key: key, conns: conns}
}

func (cli MockClient) GetConnByKey(ctx context.Context, key string) (protomfx.ConnByKeyRes, error) {
	thID, ok := cli.key[key]
	if !ok {
		return protomfx.ConnByKeyRes{}, errors.ErrAuthentication
	}

	chID, ok := cli.conns[thID]
	if !ok {
		return protomfx.ConnByKeyRes{}, errors.ErrAuthentication
	}

	conn := &protomfx.ConnByKeyRes{
		ThingID:   thID,
		ChannelID: chID,
	}

	return *conn, nil
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
func NewAuth(users map[string]string, authz map[string][]SubjectSet) protomfx.AuthServiceClient {
	return &authServiceMock{users, authz}
}

func (svc authServiceMock) Identify(ctx context.Context, in *protomfx.Token, opts ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &protomfx.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *protomfx.IssueReq, opts ...grpc.CallOption) (*protomfx.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &protomfx.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	if req.GetToken() != "token" {
		return &empty.Empty{}, errors.ErrAuthorization
	}

	return &empty.Empty{}, nil
}

func (svc authServiceMock) AssignRole(ctx context.Context, req *protomfx.AssignRoleReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	panic("not implemented")
}
