// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

const (
	topic             = "messages"
	subtopic          = "engine"
	topicWithSubtopic = topic + "/" + subtopic
	tokenTimeout      = 100 * time.Millisecond
	senmlContentType  = "application/senml+json"
)

var (
	// ErrFailedHandleMessage indicates that the message couldn't be handled.
	errFailedHandleMessage = errors.New("failed to handle mainflux message")

	pubID                                      = "pid"
	clientID1, clientID2, clientID3, clientID4 = "cid1", "cid2", "cid3", "cid4"
	data                                       = []byte("payload")
)

func TestPublisher(t *testing.T) {
	msgChan := make(chan []byte)

	// Subscribing with topic, and with subtopic, so that we can publish messages.
	client, err := newClient(address, pubID, brokerTimeout)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	token := client.Subscribe(topic, qos, func(c mqtt.Client, m mqtt.Message) {
		msgChan <- m.Payload()
	})
	if ok := token.WaitTimeout(tokenTimeout); !ok {
		assert.Fail(t, fmt.Sprintf("failed to subscribe to topic %s", topicWithSubtopic))
	}
	assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

	token = client.Subscribe(topicWithSubtopic, qos, func(c mqtt.Client, m mqtt.Message) {
		msgChan <- m.Payload()
	})
	if ok := token.WaitTimeout(tokenTimeout); !ok {
		assert.Fail(t, fmt.Sprintf("failed to subscribe to topic %s", topicWithSubtopic))
	}
	assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

	t.Cleanup(func() {
		token := client.Unsubscribe(topic, topicWithSubtopic)
		token.WaitTimeout(tokenTimeout)
		assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

		client.Disconnect(100)
	})

	cases := []struct {
		desc    string
		subject string
		payload []byte
	}{
		{
			desc:    "publish message with empty payload",
			payload: []byte{},
			subject: topic,
		},
		{
			desc:    "publish message with string payload",
			payload: data,
			subject: topic,
		},
		{
			desc:    "publish message with subtopic",
			payload: data,
			subject: topicWithSubtopic,
		},
	}
	for _, tc := range cases {
		msg := protomfx.Message{
			Publisher:   pubID,
			Subject:     tc.subject,
			Payload:     tc.payload,
			ContentType: senmlContentType,
		}

		err := pubsub.Publish(msg)
		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s\n", tc.desc, err))

		data, err := proto.Marshal(&msg)
		assert.Nil(t, err, fmt.Sprintf("%s: failed to serialize protobuf error: %s\n", tc.desc, err))

		receivedMsg := <-msgChan
		assert.Equal(t, data, receivedMsg, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, data, receivedMsg))
	}
}

func TestSubscribe(t *testing.T) {
	msgChan := make(chan protomfx.Message)

	// Creating client to Publish messages to subscribed topic.
	client, err := newClient(address, "mainflux", brokerTimeout)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	t.Cleanup(func() {
		client.Unsubscribe()
		client.Disconnect(100)
	})

	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: clientID1,
			err:      nil,
			handler:  handler{false, clientID1, msgChan},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: clientID2,
			err:      nil,
			handler:  handler{false, clientID2, msgChan},
		},
		{
			desc:     "Subscribe to an already subscribed topic with an ID",
			topic:    topic,
			clientID: clientID1,
			err:      nil,
			handler:  handler{false, clientID1, msgChan},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    topicWithSubtopic,
			clientID: clientID1,
			err:      nil,
			handler:  handler{false, clientID1, msgChan},
		},
		{
			desc:     "Subscribe to an already subscribed topic with a subtopic with an ID",
			topic:    topicWithSubtopic,
			clientID: clientID1,
			err:      nil,
			handler:  handler{false, clientID1, msgChan},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: clientID1,
			err:      messaging.ErrEmptyTopic,
			handler:  handler{false, clientID1, msgChan},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      messaging.ErrEmptyID,
			handler:  handler{false, "", msgChan},
		},
	}
	for _, tc := range cases {
		err = pubsub.Subscribe(tc.clientID, tc.topic, tc.handler)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, err, tc.err))

		if tc.err == nil {
			expectedMsg := protomfx.Message{
				Publisher: pubID,
				Subtopic:  subtopic,
				Payload:   data,
			}
			data, err := proto.Marshal(&expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("%s: failed to serialize protobuf error: %s\n", tc.desc, err))

			token := client.Publish(tc.topic, qos, false, data)
			token.WaitTimeout(tokenTimeout)
			assert.Nil(t, token.Error(), fmt.Sprintf("got unexpected error: %s", token.Error()))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, expectedMsg, receivedMsg))
		}
	}
}

func TestPubSub(t *testing.T) {
	msgChan := make(chan protomfx.Message)

	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: clientID3,
			err:      nil,
			handler:  handler{false, clientID3, msgChan},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: clientID4,
			err:      nil,
			handler:  handler{false, clientID4, msgChan},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    topicWithSubtopic,
			clientID: clientID3,
			err:      nil,
			handler:  handler{false, clientID3, msgChan},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: clientID3,
			err:      messaging.ErrEmptyTopic,
			handler:  handler{false, clientID3, msgChan},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      messaging.ErrEmptyID,
			handler:  handler{false, "", msgChan},
		},
	}
	for _, tc := range cases {
		err := pubsub.Subscribe(tc.clientID, tc.topic, tc.handler)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, err, tc.err))

		if tc.err == nil {
			// Use pubsub to subscribe to a topic, and then publish messages to that topic.
			expectedMsg := protomfx.Message{
				Publisher:   pubID,
				Subtopic:    subtopic,
				Subject:     topic,
				Payload:     data,
				ContentType: senmlContentType,
			}

			// Publish message, and then receive it on message profile.
			err := pubsub.Publish(expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s\n", tc.desc, err))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg, receivedMsg, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, expectedMsg, receivedMsg))
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	msgChan := make(chan protomfx.Message)
	topic2 := "test"
	topicWithSubtopic2 := topic2 + "/" + "subtopic"
	clientID5, clientID6, clientID7, clientID8 := "cid5", "cid6", "cid7", "cid8"

	cases := []struct {
		desc      string
		topic     string
		clientID  string
		err       error
		subscribe bool // True for subscribing and false for unsubscribing.
		handler   messaging.MessageHandler
	}{
		{
			desc:      "Subscribe to a topic with an ID",
			topic:     topic,
			clientID:  clientID5,
			err:       nil,
			subscribe: true,
			handler:   handler{false, clientID5, msgChan},
		},
		{
			desc:      "Subscribe to the same topic with a different ID",
			topic:     topic,
			clientID:  clientID6,
			err:       nil,
			subscribe: true,
			handler:   handler{false, clientID6, msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with an ID",
			topic:     topic,
			clientID:  clientID5,
			err:       nil,
			subscribe: false,
			handler:   handler{false, clientID5, msgChan},
		},
		{
			desc:      "Unsubscribe from same topic with different ID",
			topic:     topic,
			clientID:  clientID6,
			err:       nil,
			subscribe: false,
			handler:   handler{false, clientID6, msgChan},
		},
		{
			desc:      "Unsubscribe from a non-existent topic with an ID",
			topic:     "h",
			clientID:  clientID5,
			err:       messaging.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, clientID5, msgChan},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with an ID",
			topic:     topic,
			clientID:  clientID5,
			err:       messaging.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, clientID5, msgChan},
		},
		{
			desc:      "Subscribe to a topic with a subtopic with an ID",
			topic:     topicWithSubtopic,
			clientID:  clientID7,
			err:       nil,
			subscribe: true,
			handler:   handler{false, clientID7, msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with a subtopic with an ID",
			topic:     topicWithSubtopic,
			clientID:  clientID7,
			err:       nil,
			subscribe: false,
			handler:   handler{false, clientID7, msgChan},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with a subtopic with an ID",
			topic:     topicWithSubtopic,
			clientID:  clientID7,
			err:       messaging.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, clientID7, msgChan},
		},
		{
			desc:      "Unsubscribe from an empty topic with an ID",
			topic:     "",
			clientID:  clientID5,
			err:       messaging.ErrEmptyTopic,
			subscribe: false,
			handler:   handler{false, clientID5, msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with empty ID",
			topic:     topic,
			clientID:  "",
			err:       messaging.ErrEmptyID,
			subscribe: false,
			handler:   handler{false, "", msgChan},
		},
		{
			desc:      "Subscribe to a new topic with an ID",
			topic:     topic2,
			clientID:  clientID8,
			err:       nil,
			subscribe: true,
			handler:   handler{true, clientID8, msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with an ID with failing handler",
			topic:     topic2,
			clientID:  clientID8,
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, clientID8, msgChan},
		},
		{
			desc:      "Subscribe to a new topic with subtopic with an ID",
			topic:     topicWithSubtopic2,
			clientID:  clientID8,
			err:       nil,
			subscribe: true,
			handler:   handler{true, clientID8, msgChan},
		},
		{
			desc:      "Unsubscribe from a topic with subtopic with an ID with failing handler",
			topic:     topicWithSubtopic2,
			clientID:  clientID8,
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, clientID8, msgChan},
		},
	}
	for _, tc := range cases {
		switch tc.subscribe {
		case true:
			err := pubsub.Subscribe(tc.clientID, tc.topic, tc.handler)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		default:
			err := pubsub.Unsubscribe(tc.clientID, tc.topic)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		}
	}
}

type handler struct {
	fail      bool
	publisher string
	msgChan   chan protomfx.Message
}

func (h handler) Handle(msg protomfx.Message) error {
	if msg.Publisher != h.publisher {
		h.msgChan <- msg
	}
	return nil
}

func (h handler) Cancel() error {
	if h.fail {
		return errFailedHandleMessage
	}
	return nil
}
