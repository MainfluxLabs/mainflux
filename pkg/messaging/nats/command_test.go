// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishCommand(t *testing.T) {
	cp, ok := pubsub.(nats.PubSub)
	require.True(t, ok, "test harness pubsub must satisfy nats.PubSub")

	const (
		thingID  = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
		rotation = "cert-rotation"
	)
	subject := nats.GetThingCommandsSubject(thingID, rotation)

	err := pubsub.Subscribe("cmd-test-sub", subject, handler{})
	require.Nil(t, err, fmt.Sprintf("subscribe failed: %s", err))

	cmd := protomfx.Command{
		Publisher:   thingID,
		Protocol:    "certs",
		Payload:     []byte(`{"thing_id":"x","action":"rotate"}`),
		RecipientID: thingID,
	}
	err = cp.PublishCommand(subject, cmd)
	require.Nil(t, err, fmt.Sprintf("PublishCommand failed: %s", err))

	got := <-msgChan
	// Command and Message share the gogo-protobuf wire layout; the existing
	// subscriber decodes into Message. Command.RecipientID and Message.ContentType
	// occupy the same field (4), so the recipient lands in got.ContentType on decode.
	assert.Equal(t, cmd.Publisher, got.Publisher)
	assert.Equal(t, cmd.Protocol, got.Protocol)
	assert.Equal(t, cmd.RecipientID, got.ContentType)
	assert.Equal(t, cmd.Payload, got.Payload)
}
