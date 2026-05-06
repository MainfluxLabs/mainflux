// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	grpcapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	port           = 8081
	secret         = "secret"
	email          = "test@example.com"
	id             = "testID"
	orgID          = "orgID"
	adminID        = "adminID"
	adminEmail     = "admin@example.com"
	viewerID       = "viewerID"
	viewerEmail    = "viewer@example.com"
	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour
)

var (
	svc         auth.Service
	membersMock auth.OrgMembershipsRepository
)

func newService() auth.Service {
	repo := mocks.NewKeyRepository()
	roles := mocks.NewRolesRepository()
	membersMock = mocks.NewOrgMembershipsRepository()
	idProvider := uuid.NewMock()
	t := jwt.New(secret)

	return auth.New(nil, nil, nil, repo, roles, membersMock, nil, nil, idProvider, t, loginDuration, inviteDuration)
}

func startGRPCServer(svc auth.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	protomfx.RegisterAuthServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func TestIssue(t *testing.T) {
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, mocktracer.New(), time.Second)

	cases := []struct {
		desc  string
		id    string
		email string
		kind  uint32
		err   error
		code  codes.Code
	}{
		{
			desc:  "issue for user with valid token",
			id:    id,
			email: email,
			kind:  auth.LoginKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue recovery key",
			id:    id,
			email: email,
			kind:  auth.RecoveryKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue API key unauthenticated",
			id:    id,
			email: email,
			kind:  auth.APIKey,
			err:   nil,
			code:  codes.Unauthenticated,
		},
		{
			desc:  "issue for invalid key type",
			id:    id,
			email: email,
			kind:  32,
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.InvalidArgument,
		},
		{
			desc:  "issue for user that exist",
			id:    "",
			email: "",
			kind:  auth.APIKey,
			err:   status.Error(codes.Unauthenticated, "unauthenticated access"),
			code:  codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		_, err := client.Issue(context.Background(), tc.id, tc.email, tc.kind)
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	_, userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	require.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	_, recoveryToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	require.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	_, apiToken, err := svc.Issue(context.Background(), userToken, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute), IssuerID: id, Subject: email})
	require.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, mocktracer.New(), time.Second)

	cases := []struct {
		desc  string
		token string
		idt   domain.Identity
		err   error
		code  codes.Code
	}{
		{
			desc:  "identify user with user token",
			token: userToken,
			idt:   domain.Identity{Email: email, ID: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with recovery token",
			token: recoveryToken,
			idt:   domain.Identity{Email: email, ID: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with API token",
			token: apiToken,
			idt:   domain.Identity{Email: email, ID: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with invalid user token",
			token: "invalid",
			idt:   domain.Identity{},
			err:   status.Error(codes.Unauthenticated, "unauthenticated access"),
			code:  codes.Unauthenticated,
		},
		{
			desc:  "identify user with empty token",
			token: "",
			idt:   domain.Identity{},
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		idt, err := client.Identify(context.Background(), tc.token)
		if err == nil {
			assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
		}
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestAuthorize(t *testing.T) {
	_, userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	require.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	require.Nil(t, err, fmt.Sprintf("Issuing admin login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), adminID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("Assigning admin role expected to succeed: %s", err))

	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	require.Nil(t, err, fmt.Sprintf("Issuing viewer login key expected to succeed: %s", err))

	err = membersMock.Save(context.Background(), auth.OrgMembership{MemberID: viewerID, OrgID: orgID, Role: auth.Viewer})
	require.Nil(t, err, fmt.Sprintf("Saving org membership expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, mocktracer.New(), time.Second)

	cases := []struct {
		desc string
		ar   domain.AuthzReq
		code codes.Code
	}{
		{
			desc: "authorize with empty token",
			ar:   domain.AuthzReq{Token: "", Subject: auth.RootSub},
			code: codes.Unauthenticated,
		},
		{
			desc: "authorize with invalid token",
			ar:   domain.AuthzReq{Token: "invalid", Subject: auth.RootSub},
			code: codes.Unauthenticated,
		},
		{
			desc: "authorize with invalid subject",
			ar:   domain.AuthzReq{Token: userToken, Subject: "invalid"},
			code: codes.Internal,
		},
		{
			desc: "authorize with valid token, non-admin user",
			ar:   domain.AuthzReq{Token: userToken, Subject: auth.RootSub},
			code: codes.PermissionDenied,
		},
		{
			desc: "authorize with valid token, admin user",
			ar:   domain.AuthzReq{Token: adminToken, Subject: auth.RootSub},
			code: codes.OK,
		},
		{
			desc: "authorize org member with no membership",
			ar:   domain.AuthzReq{Token: userToken, Subject: auth.OrgSub, Object: orgID, Action: auth.Viewer},
			code: codes.NotFound,
		},
		{
			desc: "authorize org member with viewer role and viewer action",
			ar:   domain.AuthzReq{Token: viewerToken, Subject: auth.OrgSub, Object: orgID, Action: auth.Viewer},
			code: codes.OK,
		},
		{
			desc: "authorize org member with viewer role and editor action",
			ar:   domain.AuthzReq{Token: viewerToken, Subject: auth.OrgSub, Object: orgID, Action: auth.Editor},
			code: codes.PermissionDenied,
		},
		{
			desc: "authorize admin user via org subject",
			ar:   domain.AuthzReq{Token: adminToken, Subject: auth.OrgSub, Object: orgID, Action: auth.Owner},
			code: codes.OK,
		},
	}

	for _, tc := range cases {
		err := client.Authorize(context.Background(), tc.ar)
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}
