// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	chansPrefix      = "channels"
	channel          = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
	subtopic         = "engine"
	clientID         = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
	senmlContentType = "application/senml+json"
	senmlFormat      = "senml"
	messagesSuffix   = "messages"
)

var (
	msgChan    = make(chan protomfx.Message)
	data       = []byte("payload")
	errFailed  = errors.New("failed")
	msgProfile = &protomfx.Profile{ContentType: senmlContentType, Write: true}
)

func TestPublisher(t *testing.T) {
	format := senmlFormat + "." + messagesSuffix
	err := pubsub.Subscribe(clientID, fmt.Sprintf("%s.%s.%s", chansPrefix, channel, format), handler{})
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	err = pubsub.Subscribe(clientID, fmt.Sprintf("%s.%s.%s.%s", chansPrefix, channel, format, subtopic), handler{})
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		subtopic string
		payload  []byte
	}{
		{
			desc:    "publish message with nil payload",
			payload: nil,
		},
		{
			desc:    "publish message with string payload",
			payload: data,
		},
		{
			desc:     "publish message with subtopic",
			payload:  data,
			subtopic: subtopic,
		},
	}

	for _, tc := range cases {
		expectedMsg := protomfx.Message{
			Channel:  channel,
			Subtopic: tc.subtopic,
			Payload:  tc.payload,
			Profile:  msgProfile,
		}

		err = pubsub.Publish(expectedMsg)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		receivedMsg := <-msgChan
		assert.Equal(t, expectedMsg, receivedMsg, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, expectedMsg, receivedMsg))
	}
}

func TestPubsub(t *testing.T) {
	// Test Subscribe and Unsubscribe
	subcases := []struct {
		desc         string
		topic        string
		clientID     string
		errorMessage error
		pubsub       bool //true for subscribe and false for unsubscribe
		handler      messaging.MessageHandler
	}{
		{
			desc:         "Subscribe to a topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientid1",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to the same topic with a different ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientid2",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an already subscribed topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientid1",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from a topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientid1",
			errorMessage: nil,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from a non-existent topic with an ID",
			topic:        "h",
			clientID:     "clientid1",
			errorMessage: messaging.ErrNotSubscribed,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from the same topic with a different ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientidd2",
			errorMessage: messaging.ErrNotSubscribed,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from the same topic with a different ID not subscribed",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientidd3",
			errorMessage: messaging.ErrNotSubscribed,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from an already unsubscribed topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "clientid1",
			errorMessage: messaging.ErrNotSubscribed,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a topic with a subtopic with an ID",
			topic:        fmt.Sprintf("%s.%s.%s", chansPrefix, channel, subtopic),
			clientID:     "clientidd1",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an already subscribed topic with a subtopic with an ID",
			topic:        fmt.Sprintf("%s.%s.%s", chansPrefix, channel, subtopic),
			clientID:     "clientidd1",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from a topic with a subtopic with an ID",
			topic:        fmt.Sprintf("%s.%s.%s", chansPrefix, channel, subtopic),
			clientID:     "clientidd1",
			errorMessage: nil,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from an already unsubscribed topic with a subtopic with an ID",
			topic:        fmt.Sprintf("%s.%s.%s", chansPrefix, channel, subtopic),
			clientID:     "clientid1",
			errorMessage: messaging.ErrNotSubscribed,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty topic with an ID",
			topic:        "",
			clientID:     "clientid1",
			errorMessage: messaging.ErrEmptyTopic,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from an empty topic with an ID",
			topic:        "",
			clientID:     "clientid1",
			errorMessage: messaging.ErrEmptyTopic,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a topic with empty id",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "",
			errorMessage: messaging.ErrEmptyID,
			pubsub:       true,
			handler:      handler{false},
		},
		{
			desc:         "Unsubscribe from a topic with empty id",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel),
			clientID:     "",
			errorMessage: messaging.ErrEmptyID,
			pubsub:       false,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to another topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel+"1"),
			clientID:     "clientid3",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{true},
		},
		{
			desc:         "Subscribe to another already subscribed topic with an ID with Unsubscribe failing",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel+"1"),
			clientID:     "clientid3",
			errorMessage: errFailed,
			pubsub:       true,
			handler:      handler{true},
		},
		{
			desc:         "Subscribe to a new topic with an ID",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel+"2"),
			clientID:     "clientid4",
			errorMessage: nil,
			pubsub:       true,
			handler:      handler{true},
		},
		{
			desc:         "Unsubscribe from a topic with an ID with failing handler",
			topic:        fmt.Sprintf("%s.%s", chansPrefix, channel+"2"),
			clientID:     "clientid4",
			errorMessage: errFailed,
			pubsub:       false,
			handler:      handler{true},
		},
	}

	for _, pc := range subcases {
		if pc.pubsub == true {
			err := pubsub.Subscribe(pc.clientID, pc.topic, pc.handler)
			if pc.errorMessage == nil {
				require.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
			} else {
				assert.Equal(t, err, pc.errorMessage)
			}
		} else {
			err := pubsub.Unsubscribe(pc.clientID, pc.topic)
			if pc.errorMessage == nil {
				require.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
			} else {
				assert.Equal(t, err, pc.errorMessage)
			}
		}
	}
}

type handler struct {
	fail bool
}

func (h handler) Handle(msg protomfx.Message) error {
	msgChan <- msg
	return nil
}

func (h handler) Cancel() error {
	if h.fail {
		return errFailed
	}
	return nil
}
