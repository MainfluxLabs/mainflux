// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/mqtt/redis/cache"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const protocol = "mqtt"

const (
	LogInfoSubscribed   = "subscribed with client_id %s to topics %s"
	LogInfoUnsubscribed = "unsubscribed client_id %s from topics %s"
	LogInfoConnected    = "connected with client_id %s"
	LogInfoDisconnected = "disconnected client_id %s"
	LogInfoPublished    = "published with client_id %s to the topic %s"

	LogErrFailedConnect            = "failed to connect: "
	LogErrFailedSubscribe          = "failed to subscribe: "
	LogErrFailedUnsubscribe        = "failed to unsubscribe: "
	LogErrFailedDisconnect         = "failed to disconnect: "
	logErrFailedParseSubtopic      = "failed to parse subtopic: "
	logErrFailedCacheConnection    = "failed to cache connection: "
	logErrFailedCacheDisconnection = "failed to remove connection from cache: "
)

const (
	topicPrefixThings   = "things"
	topicPrefixGroups   = "groups"
	topicSuffixCommands = "commands"
	topicSuffixMessages = "messages"

	actionMessage = "message"
	actionCommand = "command"
)

var (
	ErrMalformedSubtopic             = errors.New("malformed subtopic")
	ErrClientNotInitialized          = errors.New("client is not initialized")
	ErrMissingClientID               = errors.New("missing client id")
	ErrMissingTopicPub               = errors.New("failed to publish due to missing topic")
	ErrMissingTopicSub               = errors.New("failed to subscribe due to missing topic")
	ErrUnauthorizedSubscriptionTopic = errors.New("unauthorized subscription topic")
	ErrUnauthorizedPublishTopic      = errors.New("unauthorized publish topic")
)

// Event implements events.Event interface
type handler struct {
	publisher messaging.Publisher
	things    protomfx.ThingsServiceClient
	service   Service
	cache     cache.ConnectionCache
	logger    logger.Logger
}

// NewHandler creates new Handler entity
func NewHandler(publisher messaging.Publisher, things protomfx.ThingsServiceClient,
	svc Service, cache cache.ConnectionCache, logger logger.Logger) session.Handler {
	return &handler{
		publisher: publisher,
		things:    things,
		service:   svc,
		cache:     cache,
		logger:    logger,
	}
}

// AuthConnect is called on device connection,
// prior to forwarding to the MQTT broker
func (h *handler) AuthConnect(c *session.Client) error {
	if c == nil {
		return ErrClientNotInitialized
	}

	if c.ID == "" {
		return ErrMissingClientID
	}

	if _, err := h.identify(c); err != nil {
		return err
	}

	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthPublish(c *session.Client, topic *string, _ *[]byte) error {
	if c == nil {
		return ErrClientNotInitialized
	}

	if topic == nil {
		return ErrMissingTopicPub
	}

	publisherID, err := h.identify(c)
	if err != nil {
		return err
	}

	return h.authorizePublish(publisherID, *topic)
}

func (h *handler) authorizePublish(publisherID, topic string) error {
	trimmed := strings.Trim(topic, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return nil
	}

	prefix, id, suffix := parts[0], parts[1], parts[2]
	if id == "" {
		return nil
	}

	switch {
	case prefix == topicPrefixThings && suffix == topicSuffixMessages:
		return h.checkThingCapability(publisherID, id, actionMessage)
	case prefix == topicPrefixThings && suffix == topicSuffixCommands:
		return h.checkThingCapability(publisherID, id, actionCommand)
	case prefix == topicPrefixGroups && suffix == topicSuffixCommands:
		return h.checkGroupMembership(publisherID, id)
	}

	return nil
}

func (h *handler) checkThingCapability(publisherID, recipientID, action string) error {
	if _, err := h.things.CanThingPerform(context.Background(), &protomfx.ThingCapabilityReq{
		PublisherID: publisherID,
		RecipientID: recipientID,
		Action:      action,
	}); err != nil {
		return errors.Wrap(ErrUnauthorizedPublishTopic, err)
	}
	return nil
}

func (h *handler) checkGroupMembership(publisherID, groupID string) error {
	grp, err := h.things.GetGroupIDByThing(context.Background(), &protomfx.ThingID{Value: publisherID})
	if err != nil {
		return err
	}
	if grp.GetValue() != groupID {
		return errors.Wrap(ErrUnauthorizedPublishTopic, fmt.Errorf("%s for group %s", groupID, publisherID))
	}
	return nil
}

// AuthSubscribe is called on device subscribe,
// prior to creating the subscription on the MQTT broker.
// It rejects subscribes to custom topics (commands/messages)
// when the topic ID does not match the client's thing or group.
func (h *handler) AuthSubscribe(c *session.Client, topics *[]string) error {
	if c == nil {
		return ErrClientNotInitialized
	}

	if topics == nil || *topics == nil {
		return ErrMissingTopicSub
	}

	thingID, err := h.identify(c)
	if err != nil {
		return err
	}

	groupID, err := h.things.GetGroupIDByThing(context.Background(), &protomfx.ThingID{Value: thingID})
	if err != nil {
		return err
	}

	for _, t := range *topics {
		if err := validateCustomTopic(t, thingID, groupID.GetValue()); err != nil {
			return err
		}
	}

	return nil
}

// Connect - after client successfully connected
func (h *handler) Connect(c *session.Client) {
	if c == nil {
		h.logger.Error(LogErrFailedConnect + (ErrClientNotInitialized).Error())
		return
	}

	h.logger.Info(fmt.Sprintf(LogInfoConnected, c.ID))
}

// Publish - after client successfully published
func (h *handler) Publish(c *session.Client, topic *string, payload *[]byte) {
	if c == nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, c.ID, *topic))

	subtopic, err := messaging.NormalizeSubtopic(*topic)
	if err != nil {
		h.logger.Error(logErrFailedParseSubtopic + err.Error())
		return
	}

	tk := &protomfx.ThingKey{
		Value: string(c.Password),
		Type:  c.Username,
	}

	pc, err := h.things.GetPubConfigByKey(context.Background(), tk)
	if err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, errors.ErrAuthentication).Error())
	}

	msg := protomfx.Message{
		Protocol: protocol,
		Subtopic: subtopic,
		Payload:  *payload,
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
	}

	if err := h.publisher.Publish(nats.GetMessagesSubject(msg.Publisher, msg.Subtopic), msg); err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
	}
}

// Subscribe - after client successfully subscribed
func (h *handler) Subscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error(LogErrFailedSubscribe + (ErrClientNotInitialized).Error())
		return
	}

	subs, err := h.getSubscriptions(c, topics)
	if err != nil {
		h.logger.Error(LogErrFailedSubscribe + err.Error())
		return
	}

	for _, s := range subs {
		if err = h.service.CreateSubscription(context.Background(), s); err != nil {
			h.logger.Error(LogErrFailedSubscribe + err.Error())
			return
		}
	}

	h.logger.Info(fmt.Sprintf(LogInfoSubscribed, c.ID, strings.Join(*topics, ",")))
}

// Unsubscribe - after client unsubscribed
func (h *handler) Unsubscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error(LogErrFailedUnsubscribe + (ErrClientNotInitialized).Error())
		return
	}

	subs, err := h.getSubscriptions(c, topics)
	if err != nil {
		h.logger.Error(LogErrFailedUnsubscribe + err.Error())
		return
	}

	for _, s := range subs {
		if err := h.service.RemoveSubscription(context.Background(), s); err != nil {
			h.logger.Error(LogErrFailedUnsubscribe + (ErrClientNotInitialized).Error())
		}
	}

	h.logger.Info(fmt.Sprintf(LogInfoUnsubscribed, c.ID, strings.Join(*topics, ",")))
}

// Disconnect - connection with broker or client lost
func (h *handler) Disconnect(c *session.Client) {
	if c == nil {
		h.logger.Error(LogErrFailedDisconnect + (ErrClientNotInitialized).Error())
		return
	}

	if err := h.cache.Disconnect(context.Background(), c.ID); err != nil {
		h.logger.Error(logErrFailedCacheDisconnection + err.Error())
	}

	h.logger.Info(fmt.Sprintf(LogInfoDisconnected, c.ID))
}

func (h *handler) identify(c *session.Client) (string, error) {
	// Use cache to avoid repeated Identify calls for the same MQTT client.
	if thingID := h.cache.RetrieveThingByClient(context.Background(), c.ID); thingID != "" {
		return thingID, nil
	}

	thingKeyReq := &protomfx.ThingKey{
		Value: string(c.Password),
		Type:  c.Username,
	}

	keyRes, err := h.things.Identify(context.Background(), thingKeyReq)
	if err != nil {
		return "", err
	}
	thingID := keyRes.GetValue()

	if err := h.cache.Connect(context.Background(), c.ID, thingID); err != nil {
		h.logger.Error(logErrFailedCacheConnection + err.Error())
	}

	return thingID, nil
}

func (h *handler) getSubscriptions(c *session.Client, topics *[]string) ([]Subscription, error) {
	thingID, err := h.identify(c)
	if err != nil {
		return nil, err
	}
	groupID, err := h.things.GetGroupIDByThing(context.Background(), &protomfx.ThingID{Value: thingID})
	if err != nil {
		return nil, err
	}

	var subs []Subscription
	for _, t := range *topics {
		sub := Subscription{
			Topic:     t,
			GroupID:   groupID.GetValue(),
			ThingID:   thingID,
			CreatedAt: float64(time.Now().UnixNano()) / 1e9,
		}
		subs = append(subs, sub)
	}

	return subs, nil
}

// validateCustomTopic enforces authorization only for topics that match
// custom patterns (things/thingID/commands, groups/groupID/commands, things/thingID/messages).
func validateCustomTopic(topic, thingID, groupID string) error {
	trimmed := strings.Trim(topic, "/")
	if trimmed == "" {
		return nil
	}

	if !strings.HasPrefix(trimmed, topicPrefixThings+"/") && !strings.HasPrefix(trimmed, topicPrefixGroups+"/") {
		return nil
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		// Forbid multi-level wildcard at ID position, e.g. "things/#", "groups/#".
		if len(parts) == 2 && parts[1] == "#" {
			return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s (wildcard not allowed)", topic))
		}
		return nil
	}

	prefix, id, suffix := parts[0], parts[1], parts[2]
	switch id {
	case "":
		return nil
	case "+", "#":
		// This catches "things/+/commands", "things/#/messages", etc.
		return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s (wildcard not allowed)", topic))
	}

	switch suffix {
	case topicSuffixCommands:
		if prefix == topicPrefixThings && id != thingID {
			return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s for thing %s", topic, thingID))
		}
		if prefix == topicPrefixGroups && id != groupID {
			return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s for group %s", topic, groupID))
		}
	case topicSuffixMessages:
		if prefix == topicPrefixThings && id != thingID {
			return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s for thing %s", topic, thingID))
		}
	}

	return nil
}
