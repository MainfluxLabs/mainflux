// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"context"
	"net"
	"sync/atomic"
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

	listener, err := coapnet.NewDTLSListener("udp", "127.0.0.1:0", coapnet.NewDTLSServerOptions(api.DTLSOptions(things, logger)...))
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
		psk      []byte
		success  bool
	}{
		{
			desc:     "publish with thing ID and derived PSK",
			identity: thingID,
			psk:      api.DerivePSK(thingKey),
			success:  true,
		},
		{
			desc:     "publish with wrong key",
			identity: thingID,
			psk:      api.DerivePSK("wrong"),
			success:  false,
		},
		{
			desc:     "publish with thing key as PSK",
			identity: thingID,
			psk:      []byte(thingKey),
			success:  false,
		},
		{
			desc:     "publish with unknown thing ID",
			identity: "unknown",
			psk:      api.DerivePSK(thingKey),
			success:  false,
		},
	}

	for _, tc := range cases {
		// The handshake can run on the session context, so the deadline goes on the session.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := dtls.Dial(addr, dtls.NewDTLSClientOptions(
			piondtls.WithPSK(func([]byte) ([]byte, error) { return tc.psk, nil }),
			piondtls.WithPSKIdentityHint([]byte(tc.identity)),
			piondtls.WithCipherSuites(piondtls.TLS_PSK_WITH_AES_128_CCM_8),
		), options.WithContext(ctx))
		if err != nil {
			cancel()
			assert.False(t, tc.success, "%s: unexpected dial error: %s", tc.desc, err)
			continue
		}

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

// natRelay forwards datagrams between a client and the server, and can switch to a
// new source port mid-session, the way a NAT rebinding does.
type natRelay struct {
	front    *net.UDPConn
	server   *net.UDPAddr
	upstream atomic.Pointer[net.UDPConn]
	client   atomic.Pointer[net.UDPAddr]
}

func startNATRelay(t *testing.T, serverAddr string) *natRelay {
	server, err := net.ResolveUDPAddr("udp", serverAddr)
	require.NoError(t, err)
	front, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)

	relay := &natRelay{front: front, server: server}
	relay.rebind(t)
	t.Cleanup(func() {
		_ = front.Close()
		_ = relay.upstream.Load().Close()
	})

	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := front.ReadFromUDP(buf)
			if err != nil {
				return
			}
			relay.client.Store(addr)
			_, _ = relay.upstream.Load().Write(buf[:n])
		}
	}()

	return relay
}

// rebind sends further datagrams to the server from a new source port.
func (relay *natRelay) rebind(t *testing.T) {
	upstream, err := net.DialUDP("udp", nil, relay.server)
	require.NoError(t, err)
	if previous := relay.upstream.Swap(upstream); previous != nil {
		_ = previous.Close()
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			n, err := upstream.Read(buf)
			if err != nil {
				return
			}
			if client := relay.client.Load(); client != nil {
				_, _ = relay.front.WriteToUDP(buf[:n], client)
			}
		}
	}()
}

func (relay *natRelay) addr() string {
	return relay.front.LocalAddr().String()
}

func TestDTLSConnectionID(t *testing.T) {
	svc := &serviceMock{keys: make(chan domain.ThingKey, 1)}
	relay := startNATRelay(t, startDTLSServer(t, svc))

	conn, err := dtls.Dial(relay.addr(), dtls.NewDTLSClientOptions(
		piondtls.WithPSK(func([]byte) ([]byte, error) { return api.DerivePSK(thingKey), nil }),
		piondtls.WithPSKIdentityHint([]byte(thingID)),
		piondtls.WithCipherSuites(piondtls.TLS_PSK_WITH_AES_128_CCM_8),
		piondtls.WithConnectionIDGenerator(piondtls.OnlySendCIDGenerator()),
	))
	require.NoError(t, err)
	defer conn.Close()

	publish := func(desc string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		resp, err := conn.Post(ctx, "/things/"+thingID+"/messages", message.TextPlain, bytes.NewReader([]byte(`{"v":1}`)))
		require.NoError(t, err, desc)
		assert.Equal(t, codes.Changed, resp.Code(), desc)
		assert.Equal(t, domain.ThingKey{Value: thingKey}, <-svc.keys, desc)
	}

	publish("publish before address change")
	relay.rebind(t)
	publish("publish on the same session after address change")
}

func TestDerivePSK(t *testing.T) {
	// Matches: echo -n <key> | sha256sum | cut -c1-32
	assert.Equal(t, []byte("6395761e7cb4afc7a85bc2e7910dbe3e"), api.DerivePSK(thingKey))
}
