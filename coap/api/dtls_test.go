// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/coap"
	"github.com/MainfluxLabs/mainflux/coap/api"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	piondtls "github.com/pion/dtls/v3"
	"github.com/plgd-dev/go-coap/v3/dtls"
	dtlsserver "github.com/plgd-dev/go-coap/v3/dtls/server"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	coapnet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID  = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	thingKey = "f6f7a7d4-5e44-4a4a-a1f5-2f4b5e1c6f88"
)

// serviceMock records the key Publish receives; it implements no other methods.
type serviceMock struct {
	coap.Service
	keys chan domain.ThingKey
}

func (svc *serviceMock) Publish(_ context.Context, key domain.ThingKey, _ protomfx.Message) error {
	svc.keys <- key
	return nil
}

func startDTLSServer(t *testing.T, svc coap.Service) string {
	logger := logger.NewMock()
	thing := domain.Thing{ID: thingID, Key: thingKey}
	things := mocks.NewThingsServiceClient(nil, map[string]domain.Thing{thingID: thing}, nil)

	listener, err := coapnet.NewDTLSListener("udp", "127.0.0.1:0", coapnet.NewDTLSServerOptions(
		piondtls.WithPSK(api.PSK(things, logger)),
		piondtls.WithCipherSuites(piondtls.TLS_PSK_WITH_AES_128_CCM_8),
	))
	require.NoError(t, err)

	server := dtlsserver.New(
		options.WithMux(api.MakeCoAPHandler(svc, logger)),
		options.WithOnNewConn(api.OnNewDTLSConn(things, logger)),
	)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func TestDTLS(t *testing.T) {
	svc := &serviceMock{keys: make(chan domain.ThingKey, 1)}
	addr := startDTLSServer(t, svc)

	cases := []struct {
		desc     string
		identity string
		key      string
		success  bool
	}{
		{
			desc:     "publish with thing ID and key as PSK",
			identity: thingID,
			key:      thingKey,
			success:  true,
		},
		{
			desc:     "publish with wrong key",
			identity: thingID,
			key:      "wrong",
			success:  false,
		},
		{
			desc:     "publish with unknown thing ID",
			identity: "unknown",
			key:      thingKey,
			success:  false,
		},
	}

	for _, tc := range cases {
		conn, err := dtls.Dial(addr, dtls.NewDTLSClientOptions(
			piondtls.WithPSK(func([]byte) ([]byte, error) { return []byte(tc.key), nil }),
			piondtls.WithPSKIdentityHint([]byte(tc.identity)),
			piondtls.WithCipherSuites(piondtls.TLS_PSK_WITH_AES_128_CCM_8),
		))
		if err != nil {
			assert.False(t, tc.success, "%s: unexpected dial error: %s", tc.desc, err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := conn.Post(ctx, "/things/"+thingID+"/messages", message.TextPlain, bytes.NewReader([]byte(`{"v":1}`)))
		cancel()
		_ = conn.Close()

		if !tc.success {
			assert.Error(t, err, "%s: expected handshake to fail", tc.desc)
			continue
		}
		require.NoError(t, err, tc.desc)
		assert.Equal(t, codes.Changed, resp.Code(), tc.desc)
		assert.Equal(t, domain.ThingKey{Value: thingKey}, <-svc.keys, tc.desc)
	}
}
