// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/messaging/rabbitmq"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishCommand(t *testing.T) {
	cp, ok := pubsub.(rabbitmq.PubSub)
	require.True(t, ok, "test harness pubsub must satisfy rabbitmq.PubSub")

	const (
		thingID  = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
		rotation = "cert-rotation"
	)
	subject := fmt.Sprintf("things.%s.commands.%s", thingID, rotation)

	conn, ch, err := newConn()
	require.Nil(t, err, fmt.Sprintf("connect failed: %s", err))
	t.Cleanup(func() {
		ch.Close()
		conn.Close()
	})

	deliveries := subscribe(t, ch, subject)
	go rabbitHandler(deliveries, handler{})

	cmd := protomfx.Command{
		Publisher:   thingID,
		Protocol:    "certs",
		Payload:     []byte(`{"thing_id":"x","action":"rotate"}`),
		ContentType: "application/json",
	}
	err = cp.PublishCommand(subject, cmd)
	require.Nil(t, err, fmt.Sprintf("PublishCommand failed: %s", err))

	got := <-msgChan
	// Command and Message share the gogo-protobuf wire layout; the subscriber
	// decodes into Message. Confirm the fields survive the round-trip and that
	// the routing key was honored (otherwise the subscriber sees nothing).
	assert.Equal(t, cmd.Publisher, got.Publisher)
	assert.Equal(t, cmd.Protocol, got.Protocol)
	assert.Equal(t, cmd.ContentType, got.ContentType)
	assert.Equal(t, cmd.Payload, got.Payload)
}
