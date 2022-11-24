package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceClient)(nil)

type authServiceClient struct {
	users map[string]string
}

// NewAuthServiceClient creates mock of auth service.
func NewAuthServiceClient(users map[string]string) mainflux.AuthServiceClient {
	return &authServiceClient{users}
}

func (svc authServiceClient) ListPolicies(ctx context.Context, in *mainflux.ListPoliciesReq, opts ...grpc.CallOption) (*mainflux.ListPoliciesRes, error) {
	panic("not implemented")
}

func (svc authServiceClient) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc *authServiceClient) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return new(mainflux.Token), nil
}

func (svc *authServiceClient) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	panic("not implemented")
}

func (svc authServiceClient) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	panic("not implemented")
}

func (svc authServiceClient) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	panic("not implemented")
}

func (svc *authServiceClient) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	panic("not implemented")
}

func (svc *authServiceClient) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}
