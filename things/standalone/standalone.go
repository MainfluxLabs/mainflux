// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var errUnsupported = errors.New("not supported in standalone mode")

var _ protomfx.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(email, token string) protomfx.AuthServiceClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Issue(ctx context.Context, req *protomfx.IssueReq, opts ...grpc.CallOption) (*protomfx.Token, error) {
	if repo.token != req.GetEmail() {
		return nil, errors.ErrAuthentication
	}

	return &protomfx.Token{Value: repo.token}, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, token *protomfx.Token, opts ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	if repo.token != token.GetValue() {
		return nil, errors.ErrAuthentication
	}

	return &protomfx.UserIdentity{Id: repo.email, Email: repo.email}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}

func (repo singleUserRepo) AssignRole(ctx context.Context, req *protomfx.AssignRoleReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}

func (repo singleUserRepo) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	return &protomfx.RetrieveRoleRes{}, errUnsupported
}
