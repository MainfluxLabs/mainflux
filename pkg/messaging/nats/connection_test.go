// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	broker "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streamName = "mainflux"

func TestEnsureStream(t *testing.T) {
	conn, err := broker.Connect(address)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	t.Cleanup(conn.Close)

	js, err := conn.JetStream()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	info, err := js.StreamInfo(streamName)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	// Simulate a stream created before the ota subject existed.
	config := info.Config
	config.Subjects = slices.DeleteFunc(slices.Clone(config.Subjects), func(subject string) bool {
		return subject == nats.SubjectOTA
	})
	_, err = js.UpdateStream(&config)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	pub, err := nats.NewPublisher(address)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	t.Cleanup(func() { pub.Close() })

	info, err = js.StreamInfo(streamName)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.Contains(t, info.Config.Subjects, nats.SubjectOTA, "missing subject must be added to an existing stream")
}
