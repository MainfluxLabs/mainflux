// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	grpcapi "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port        = 8081
	secret      = "secret"
	email       = "test@example.com"
	id          = "testID"
	description = "Description"

	numOfThings = 5
	numOfUsers  = 5

	authoritiesObj = "authorities"
	memberRelation = "member"
	loginDuration  = 30 * time.Minute
)

var svc auth.Service

func newService() auth.Service {
	repo := mocks.NewKeyRepository()
	idProvider := uuid.NewMock()

	mockAuthzDB := map[string][]mocks.MockSubjectSet{}
	mockAuthzDB[id] = append(mockAuthzDB[id], mocks.MockSubjectSet{Object: authoritiesObj, Relation: memberRelation})

	t := jwt.New(secret)

	return auth.New(nil, repo, idProvider, t, loginDuration, email)
}

func startGRPCServer(svc auth.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterAuthServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func TestIssue(t *testing.T) {
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

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
		_, err := client.Issue(context.Background(), &mainflux.IssueReq{Id: tc.id, Email: tc.email, Type: tc.kind})
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	_, recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	_, apiSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc  string
		token string
		idt   mainflux.UserIdentity
		err   error
		code  codes.Code
	}{
		{
			desc:  "identify user with user token",
			token: loginSecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with recovery token",
			token: recoverySecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with API token",
			token: apiSecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with invalid user token",
			token: "invalid",
			idt:   mainflux.UserIdentity{},
			err:   status.Error(codes.Unauthenticated, "unauthenticated access"),
			code:  codes.Unauthenticated,
		},
		{
			desc:  "identify user with empty token",
			token: "",
			idt:   mainflux.UserIdentity{},
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		idt, err := client.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		if idt != nil {
			assert.Equal(t, tc.idt, *idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, *idt))
		}
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

/* TODO: Finish tests when the method is finished
func TestAuthorize(t *testing.T) {
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc     string
		token    string
		subject  string
		object   string
		relation string
		ar       mainflux.AuthorizeRes
		err      error
		code     codes.Code
	}{
		{
			desc:     "authorize user with authorized token",
			token:    loginSecret,
			subject:  id,
			object:   authoritiesObj,
			relation: memberRelation,
			ar:       mainflux.AuthorizeRes{Authorized: true},
			err:      nil,
			code:     codes.OK,
		},
		{
			desc:     "authorize user with unauthorized relation",
			token:    loginSecret,
			subject:  id,
			object:   authoritiesObj,
			relation: "unauthorizedRelation",
			ar:       mainflux.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with unauthorized object",
			token:    loginSecret,
			subject:  id,
			object:   "unauthorizedobject",
			relation: memberRelation,
			ar:       mainflux.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with unauthorized subject",
			token:    loginSecret,
			subject:  "unauthorizedSubject",
			object:   authoritiesObj,
			relation: memberRelation,
			ar:       mainflux.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.PermissionDenied,
		},
		{
			desc:     "authorize user with invalid ACL",
			token:    loginSecret,
			subject:  "",
			object:   "",
			relation: "",
			ar:       mainflux.AuthorizeRes{Authorized: false},
			err:      nil,
			code:     codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		ar, err := client.Authorize(context.Background(), &mainflux.AuthorizeReq{Sub: tc.subject, Obj: tc.object, Act: tc.relation})
		if ar != nil {
			assert.Equal(t, tc.ar, *ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ar, *ar))
		}

		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}*/

func TestAddPolicy(t *testing.T) {
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	groupAdminObj := "groupadmin"

	cases := []struct {
		desc     string
		token    string
		subject  string
		object   string
		relation string
		ar       mainflux.AddPolicyRes
		err      error
		code     codes.Code
	}{
		{
			desc:     "add groupadmin policy to user",
			token:    loginSecret,
			subject:  id,
			object:   groupAdminObj,
			relation: memberRelation,
			ar:       mainflux.AddPolicyRes{Authorized: true},
			err:      nil,
			code:     codes.OK,
		},
		{
			desc:     "add policy to user with invalid ACL",
			token:    loginSecret,
			subject:  "",
			object:   "",
			relation: "",
			ar:       mainflux.AddPolicyRes{Authorized: false},
			err:      nil,
			code:     codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		apr, err := client.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Sub: tc.subject, Obj: tc.object, Act: tc.relation})
		if apr != nil {
			assert.Equal(t, tc.ar, *apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ar, *apr))
		}

		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestDeletePolicy(t *testing.T) {
	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	readRelation := "read"
	thingID := "thing"

	apr, err := client.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Sub: id, Obj: thingID, Act: readRelation})
	assert.Nil(t, err, fmt.Sprintf("Adding read policy to user expected to succeed: %s", err))
	assert.True(t, apr.GetAuthorized(), fmt.Sprintf("Adding read policy expected to make user authorized, expected %v got %v", true, apr.GetAuthorized()))

	cases := []struct {
		desc     string
		token    string
		subject  string
		object   string
		relation string
		dpr      *mainflux.DeletePolicyRes
		code     codes.Code
	}{
		{
			desc:     "delete valid policy",
			token:    loginSecret,
			subject:  id,
			object:   thingID,
			relation: readRelation,
			dpr:      &mainflux.DeletePolicyRes{Deleted: true},
			code:     codes.OK,
		},
		{
			desc:     "delete invalid policy",
			token:    loginSecret,
			subject:  "",
			object:   "",
			relation: "",
			dpr:      &mainflux.DeletePolicyRes{Deleted: false},
			code:     codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		dpr, err := client.DeletePolicy(context.Background(), &mainflux.DeletePolicyReq{Sub: tc.subject, Obj: tc.object, Act: tc.relation})
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
		assert.Equal(t, tc.dpr.GetDeleted(), dpr.GetDeleted(), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.dpr.GetDeleted(), dpr.GetDeleted()))
	}
}
