package mqtt_test

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/mqtt/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mproxy/pkg/session"
	"github.com/stretchr/testify/assert"
)

const (
	thingID      = "513d02d2-16c1-4f23-98be-9e12f8fee898"
	groupID      = "9e12f8fe-e89b-a456-12d3-513d02d21212"
	recipientID  = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	otherThingID = "11111111-2222-3333-4444-555555555555"
	otherGroupID = "66666666-7777-8888-9999-000000000000"
	clientID     = "ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"
	password     = "cccccccc-dddd-eeee-ffff-aaaaaaaaaaaa"
	subtopic     = "test-subtopic"
)

var (
	topic        = "/messages"
	payload      = []byte("[{'n':'test-name', 'v': 1.2}]")
	topics       = []string{topic}
	//Test log messages for cases the handler does not provide a return value.
	logBuffer     = bytes.Buffer{}
	sessionClient = session.Client{
		ID:       clientID,
		Username: things.KeyTypeInternal,
		Password: []byte(password),
	}
)

func TestAuthConnect(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		err     error
		session *session.Client
	}{
		{
			desc:    "connect without active session",
			err:     mqtt.ErrClientNotInitialized,
			session: nil,
		},
		{
			desc: "connect without clientID",
			err:  mqtt.ErrMissingClientID,
			session: &session.Client{
				ID:       "",
				Username: things.KeyTypeInternal,
				Password: []byte(password),
			},
		},
		{
			desc: "connect with invalid password",
			err:  errors.ErrAuthentication,
			session: &session.Client{
				ID:       clientID,
				Username: things.KeyTypeInternal,
				Password: []byte(""),
			},
		},
		{
			desc:    "connect with valid username and password",
			err:     nil,
			session: &sessionClient,
		},
	}

	for _, tc := range cases {
		err := handler.AuthConnect(tc.session)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAuthPublish(t *testing.T) {
	handler := newHandler()

	cases := []struct {
		desc    string
		client  *session.Client
		err     error
		topic   *string
		payload []byte
	}{
		{
			desc:    "publish with inactive client",
			client:  nil,
			err:     mqtt.ErrClientNotInitialized,
			topic:   &topic,
			payload: payload,
		},
		{
			desc:    "publish without topic",
			client:  &sessionClient,
			err:     mqtt.ErrMissingTopic,
			topic:   nil,
			payload: payload,
		},
		{
			desc:    "publish successfully",
			client:  &sessionClient,
			err:     nil,
			topic:   &topic,
			payload: payload,
		},
		{
			desc:    "publish to own thing messages topic",
			client:  &sessionClient,
			err:     nil,
			topic:   strPtr("things/" + thingID + "/messages"),
			payload: payload,
		},
		{
			desc:    "publish to recipient thing messages topic in same group",
			client:  &sessionClient,
			err:     nil,
			topic:   strPtr("things/" + recipientID + "/messages"),
			payload: payload,
		},
		{
			desc:    "publish to recipient thing commands topic in same group",
			client:  &sessionClient,
			err:     nil,
			topic:   strPtr("things/" + recipientID + "/commands"),
			payload: payload,
		},
		{
			desc:    "publish to thing in different group",
			client:  &sessionClient,
			err:     mqtt.ErrUnauthorizedPublishTopic,
			topic:   strPtr("things/" + otherThingID + "/commands"),
			payload: payload,
		},
		{
			desc:    "publish to own group commands topic",
			client:  &sessionClient,
			err:     nil,
			topic:   strPtr("groups/" + groupID + "/commands"),
			payload: payload,
		},
		{
			desc:    "publish to different group commands topic",
			client:  &sessionClient,
			err:     mqtt.ErrUnauthorizedPublishTopic,
			topic:   strPtr("groups/" + otherGroupID + "/commands"),
			payload: payload,
		},
		{
			// Sensors are in the group but have no command capability.
			desc: "sensor publishes to own group commands topic",
			client: &session.Client{
				ID:       recipientID,
				Username: things.KeyTypeInternal,
				Password: []byte(recipientID),
			},
			err:     mqtt.ErrUnauthorizedPublishTopic,
			topic:   strPtr("groups/" + groupID + "/commands"),
			payload: payload,
		},
	}

	for _, tc := range cases {
		err := handler.AuthPublish(tc.client, tc.topic, &tc.payload)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func strPtr(s string) *string { return &s }

func TestAuthSubscribe(t *testing.T) {
	handler := newHandler()

	unauthorizedThingTopic := []string{"things/other-id/commands/led/room"}
	unauthorizedGroupTopic := []string{"groups/other-group/commands/fw/update"}
	thingWildcardTopic := []string{"things/+/messages"}
	multiLevelWildcardTopic := []string{"things/#"}

	cases := []struct {
		desc   string
		client *session.Client
		err    error
		topic  *[]string
	}{
		{
			desc:   "subscribe without active session",
			client: nil,
			err:    mqtt.ErrClientNotInitialized,
			topic:  &topics,
		},
		{
			desc:   "subscribe without topics",
			client: &sessionClient,
			err:    mqtt.ErrMissingTopic,
			topic:  nil,
		},
		{
			desc:   "subscribe with active session and valid topics",
			client: &sessionClient,
			err:    nil,
			topic:  &topics,
		},
		{
			desc:   "subscribe to another thing's commands topic",
			client: &sessionClient,
			err:    mqtt.ErrUnauthorizedSubscriptionTopic,
			topic:  &unauthorizedThingTopic,
		},
		{
			desc:   "subscribe to another group's commands topic",
			client: &sessionClient,
			err:    mqtt.ErrUnauthorizedSubscriptionTopic,
			topic:  &unauthorizedGroupTopic,
		},
		{
			desc:   "subscribe to thing messages with single-level wildcard",
			client: &sessionClient,
			err:    mqtt.ErrUnauthorizedSubscriptionTopic,
			topic:  &thingWildcardTopic,
		},
		{
			desc:   "subscribe to things with multi-level wildcard",
			client: &sessionClient,
			err:    mqtt.ErrUnauthorizedSubscriptionTopic,
			topic:  &multiLevelWildcardTopic,
		},
	}

	for _, tc := range cases {
		err := handler.AuthSubscribe(tc.client, tc.topic)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc   string
		client *session.Client
		logMsg string
	}{
		{
			desc:   "connect without active session",
			client: nil,
			logMsg: errors.Wrap(mqtt.ErrFailedConnect, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:   "connect with active session",
			client: &sessionClient,
			logMsg: fmt.Sprintf("client_id %s connected", clientID),
		},
	}

	for _, tc := range cases {
		handler.Connect(tc.client)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestPublish(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	malformedSubtopics := topic + "/" + subtopic + "%"
	wrongCharSubtopics := topic + "/" + subtopic + ">"
	validSubtopic := topic + "/" + subtopic

	cases := []struct {
		desc    string
		client  *session.Client
		topic   string
		payload []byte
		logMsg  string
	}{
		{
			desc:    "publish without active session",
			client:  nil,
			topic:   topic,
			payload: payload,
			logMsg:  mqtt.ErrClientNotInitialized.Error(),
		},
		{
			desc:    "publish with malformed subtopic",
			client:  &sessionClient,
			topic:   malformedSubtopics,
			payload: payload,
			logMsg:  messaging.ErrMalformedSubtopic.Error(),
		},
		{
			desc:    "publish with subtopic containing wrong character",
			client:  &sessionClient,
			topic:   wrongCharSubtopics,
			payload: payload,
			logMsg:  messaging.ErrMalformedSubtopic.Error(),
		},
		{
			desc:    "publish with subtopic",
			client:  &sessionClient,
			topic:   validSubtopic,
			payload: payload,
			logMsg:  subtopic,
		},
		{
			desc:    "publish without subtopic",
			client:  &sessionClient,
			topic:   topic,
			payload: payload,
			logMsg:  "",
		},
	}

	for _, tc := range cases {
		handler.Publish(tc.client, &tc.topic, &tc.payload)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestSubscribe(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc   string
		client *session.Client
		topic  []string
		logMsg string
	}{
		{
			desc:   "subscribe without active session",
			client: nil,
			topic:  topics,
			logMsg: errors.Wrap(mqtt.ErrFailedSubscribe, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:   "subscribe with valid session and topics",
			client: &sessionClient,
			topic:  topics,
			logMsg: fmt.Sprintf("client_id %s subscribed to topics %s", clientID, topics[0]),
		},
	}

	for _, tc := range cases {
		handler.Subscribe(tc.client, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestUnsubscribe(t *testing.T) {
	handler := newHandler()
	handler.Subscribe(&sessionClient, &topics)
	logBuffer.Reset()

	cases := []struct {
		desc   string
		client *session.Client
		topic  []string
		logMsg string
	}{
		{
			desc:   "unsubscribe without active session",
			client: nil,
			topic:  topics,
			logMsg: errors.Wrap(mqtt.ErrFailedUnsubscribe, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:   "unsubscribe with valid session and topics",
			client: &sessionClient,
			topic:  topics,
			logMsg: fmt.Sprintf("client_id %s unsubscribed from topics %s", clientID, topics[0]),
		},
	}

	for _, tc := range cases {
		handler.Unsubscribe(tc.client, &tc.topic)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func TestDisconnect(t *testing.T) {
	handler := newHandler()
	logBuffer.Reset()

	cases := []struct {
		desc   string
		client *session.Client
		topic  []string
		logMsg string
	}{
		{
			desc:   "disconnect without active session",
			client: nil,
			topic:  topics,
			logMsg: errors.Wrap(mqtt.ErrFailedDisconnect, mqtt.ErrClientNotInitialized).Error(),
		},
		{
			desc:   "disconnect with valid session",
			client: &sessionClient,
			topic:  topics,
			logMsg: fmt.Sprintf("client_id %s disconnected", clientID),
		},
	}

	for _, tc := range cases {
		handler.Disconnect(tc.client)
		assert.Contains(t, logBuffer.String(), tc.logMsg)
	}
}

func newHandler() session.Handler {
	logger, err := logger.New(&logBuffer, "debug")
	if err != nil {
		log.Fatalf("failed to create logger: %s", err)
	}

	thingsClient := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			password:     {ID: thingID, GroupID: groupID, Type: things.ThingTypeController},
			thingID:      {ID: thingID, GroupID: groupID, Type: things.ThingTypeController},
			recipientID:  {ID: recipientID, GroupID: groupID, Type: things.ThingTypeSensor},
			otherThingID: {ID: otherThingID, GroupID: otherGroupID, Type: things.ThingTypeSensor},
		},
		map[string]things.Group{
			password: {ID: groupID},
		},
	)

	return mqtt.NewHandler(pkgmocks.NewPublisher(), thingsClient, newService(), mocks.NewCache(), logger)
}
