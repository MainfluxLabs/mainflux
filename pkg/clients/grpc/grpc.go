// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"fmt"
	"os"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/clients"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Connect(cfg clients.Config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.ClientTLS {
		if cfg.CaCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.CaCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create tls credentials: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		opts = append(opts, grpc.WithInsecure())
		logger.Info("gRPC communication is not encrypted")
	}

	conn, err := grpc.Dial(cfg.URL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s service: %s", cfg.ClientName, err))
		os.Exit(1)
	}

	return conn
}
