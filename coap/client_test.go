// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"net"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silentObserver returns an observer whose peer receives datagrams but never answers,
// like a device that is gone, and a channel with the CoAP type of each datagram it gets.
func silentObserver(t *testing.T) (Client, <-chan message.Type) {
	peer, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)

	types := make(chan message.Type, 16)
	go func() {
		buf := make([]byte, 1500)
		for {
			n, _, err := peer.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n > 0 {
				types <- message.Type((buf[0] >> 4) & 0x3)
			}
		}
	}()

	conn, err := udp.Dial(peer.LocalAddr().String())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
		_ = peer.Close()
	})

	return NewClient(conn, message.Token("token"), logger.NewMock()), types
}

func TestCancelSilentObserver(t *testing.T) {
	observer, types := silentObserver(t)

	done := make(chan error, 1)
	go func() { done <- observer.Cancel() }()

	select {
	case err := <-done:
		assert.NoError(t, err, "cancel: unexpected error")
	case <-time.After(2 * time.Second):
		t.Fatal("cancel: waited for an ACK from a silent observer")
	}
	assert.Equal(t, message.NonConfirmable, <-types,
		"cancel: expected a non-confirmable final notification")
}

func TestNotifySilentObserver(t *testing.T) {
	previous := notificationTimeout
	notificationTimeout = 500 * time.Millisecond
	defer func() { notificationTimeout = previous }()

	observer, types := silentObserver(t)

	done := make(chan error, 1)
	cmd := protomfx.Command{Payload: []byte(`{"led":1}`)}
	go func() { done <- observer.HandleCommand("", cmd) }()

	select {
	case err := <-done:
		assert.Error(t, err, "notify: expected an error without an ACK")
	case <-time.After(3 * time.Second):
		t.Fatal("notify: kept waiting for an ACK after the timeout")
	}
	assert.Equal(t, message.Confirmable, <-types, "notify: expected a confirmable notification")
}
