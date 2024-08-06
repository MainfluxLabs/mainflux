// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	grpcapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	thmocks "github.com/MainfluxLabs/mainflux/things/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/opentracing/opentracing-go/mocktracer"
	"google.golang.org/grpc"
)

const (
	port     = 8080
	wrong    = "wrong"
	email    = "user@example.com"
	token    = email
	password = "password"
)

var usersList = []users.User{{Email: email, Password: password}}

var svc things.Service

func TestMain(m *testing.M) {
	startServer()
	code := m.Run()
	os.Exit(code)
}

func startServer() {
	svc = newService(map[string]string{token: email})
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	protomfx.RegisterThingsServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func newService(tokens map[string]string) things.Service {
	auth := mocks.NewAuthService("", usersList)
	conns := make(chan thmocks.Connection)
	thingsRepo := thmocks.NewThingRepository(conns)
	channelsRepo := thmocks.NewChannelRepository(thingsRepo, conns)
	groupsRepo := thmocks.NewGroupRepository()
	rolesRepo := thmocks.NewRolesRepository()
	chanCache := thmocks.NewChannelCache()
	thingCache := thmocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, nil, thingsRepo, channelsRepo, groupsRepo, rolesRepo, chanCache, thingCache, idProvider)
}
