// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var errUnsupported = errors.New("not supported in standalone mode")

var _ mainflux.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(email, token string) mainflux.AuthServiceClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Issue(ctx context.Context, req *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if repo.token != req.GetEmail() {
		return nil, errors.ErrAuthentication
	}

	return &mainflux.Token{Value: repo.token}, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, token *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if repo.token != token.GetValue() {
		return nil, errors.ErrAuthentication
	}

	return &mainflux.UserIdentity{Id: repo.email, Email: repo.email}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}

func (repo singleUserRepo) CanAccessGroup(ctx context.Context, in *mainflux.AccessGroupReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}

func (repo singleUserRepo) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	return &mainflux.MembersRes{}, errUnsupported
}

func (repo singleUserRepo) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}
