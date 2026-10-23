// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package modbus

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	thingsSvc := pkgmocks.NewThingsServiceClient(nil, nil, nil)
	svc := New(thingsSvc, pkgmocks.NewPublisher(), nil, uuid.NewMock(), logger.NewMock()).(*clientsService)

	open, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	defer open.Close()
	host, openPort, err := net.SplitHostPort(open.Addr().String())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	closed, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, closedPort, err := net.SplitHostPort(closed.Addr().String())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	closed.Close()

	cases := []struct {
		desc    string
		port    string
		wantErr bool
	}{
		{
			desc:    "connect to available device",
			port:    openPort,
			wantErr: false,
		},
		{
			desc:    "connect to unavailable device",
			port:    closedPort,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		_, mu, err := svc.connect(context.Background(), host, tc.port, 1)
		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		mu.Unlock()
	}
}
