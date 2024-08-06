// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	grpcauth "github.com/MainfluxLabs/mainflux/auth/api/grpc"
	"github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/servers"
	"github.com/MainfluxLabs/mainflux/things"
	grpcthings "github.com/MainfluxLabs/mainflux/things/api/grpc"
	"github.com/MainfluxLabs/mainflux/users"
	grpcusers "github.com/MainfluxLabs/mainflux/users/api/grpc"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Start(ctx context.Context, tracer opentracing.Tracer, svc interface{}, cfg servers.Config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.Port)
	errCh := make(chan error)

	listener, err := net.Listen("tcp", p)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.Port, err)
	}

	var server *grpc.Server
	switch {
	case cfg.ServerCert != "" || cfg.ServerKey != "":
		creds, err := credentials.NewServerTLSFromFile(cfg.ServerCert, cfg.ServerKey)
		if err != nil {
			return fmt.Errorf("failed to load auth certificates: %w", err)
		}
		logger.Info(fmt.Sprintf("%s gRPC service started using https on port %s with cert %s key %s", cfg.ServerName, cfg.Port, cfg.ServerCert, cfg.ServerKey))
		server = grpc.NewServer(grpc.Creds(creds))
	default:
		logger.Info(fmt.Sprintf("%s gRPC service started using http on port %s", cfg.ServerName, cfg.Port))
		server = grpc.NewServer()
	}

	switch v := svc.(type) {
	case things.Service:
		protomfx.RegisterThingsServiceServer(server, grpcthings.NewServer(tracer, v))
	case users.Service:
		protomfx.RegisterUsersServiceServer(server, grpcusers.NewServer(tracer, v))
	case auth.Service:
		protomfx.RegisterAuthServiceServer(server, grpcauth.NewServer(tracer, v))
	default:
		return fmt.Errorf("unknown service: %s", cfg.ServerName)
	}

	logger.Info(fmt.Sprintf("%s gRPC service started, exposed port %s", cfg.ServerName, cfg.Port))
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		c := make(chan bool)
		go func() {
			defer close(c)
			server.GracefulStop()
		}()
		select {
		case <-c:
		case <-time.After(cfg.StopWaitTime):
		}
		logger.Info(fmt.Sprintf("%s gRPC service shutdown at %s", cfg.ServerName, p))
		return nil
	case err := <-errCh:
		return err
	}
}
