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

const (
	protocol = "mqtt"

	topicPrefixThings   = "things"
	topicPrefixGroups   = "groups"
	topicSuffixCommands = "commands"
	topicSuffixMessages = "messages"
)

var (
	ErrClientNotInitialized          = errors.New("client is not initialized")
	ErrMissingClientID               = errors.New("missing client id")
	ErrMissingTopic                  = errors.New("missing topic")
	ErrUnauthorizedSubscriptionTopic = errors.New("unauthorized subscription topic")
	ErrUnauthorizedPublishTopic      = errors.New("unauthorized publish topic")

	errFailedConnect            = errors.New("failed to connect")
	errFailedDisconnect         = errors.New("failed to disconnect")
	errFailedSubscribe          = errors.New("failed to subscribe")
	errFailedUnsubscribe        = errors.New("failed to unsubscribe")
	errFailedParseSubtopic      = errors.New("failed to parse subtopic")
	errFailedCacheConnection    = errors.New("failed to cache connection")
	errFailedCacheDisconnection = errors.New("failed to remove connection from cache")
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
		return ErrMissingTopic
	}

	publisherID, err := h.identify(c)
	if err != nil {
		return err
	}

	return h.authorizePublish(publisherID, *topic)
}

func (h *handler) authorizePublish(publisherID, topic string) error {
	// Reject leading-slash variants of the custom topic patterns.
	if strings.HasPrefix(topic, "/"+topicPrefixThings+"/") || strings.HasPrefix(topic, "/"+topicPrefixGroups+"/") {
		return errors.Wrap(ErrUnauthorizedPublishTopic, fmt.Errorf("%s (leading slash not allowed)", topic))
	}

	parts := strings.Split(topic, "/")
	if len(parts) < 3 {
		return nil
	}

	prefix, id, suffix := parts[0], parts[1], parts[2]
	if id == "" {
		return nil
	}

	// Messages are unrestricted — any authenticated thing may publish to any messages topic.
	// Commands carry authority and require explicit authorization.
	var err error
	switch {
	case prefix == topicPrefixThings && suffix == topicSuffixCommands:
		err = h.authorizeThingCommand(publisherID, id)
	case prefix == topicPrefixGroups && suffix == topicSuffixCommands:
		err = h.authorizeGroupCommand(publisherID, id)
	}

	if err != nil {
		return errors.Wrap(ErrUnauthorizedPublishTopic, fmt.Errorf("%s for publisher %s", topic, publisherID))
	}
	return nil
}

func (h *handler) authorizeThingCommand(publisherID, recipientID string) error {
	if _, err := h.things.CanThingCommand(context.Background(), &protomfx.ThingCommandReq{
		PublisherID: publisherID,
		RecipientID: recipientID,
	}); err != nil {
		return errors.ErrAuthorization
	}
	return nil
}

func (h *handler) authorizeGroupCommand(publisherID, groupID string) error {
	if _, err := h.things.CanThingGroupCommand(context.Background(), &protomfx.ThingGroupCommandReq{
		PublisherID: publisherID,
		GroupID:     groupID,
	}); err != nil {
		return errors.ErrAuthorization
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
		return ErrMissingTopic
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
		h.logger.Error(errors.Wrap(errFailedConnect, ErrClientNotInitialized).Error())
		return
	}

	h.logger.Info(fmt.Sprintf("client_id %s connected", c.ID))
}

// Publish - after client successfully published
func (h *handler) Publish(c *session.Client, topic *string, payload *[]byte) {
	if c == nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf("client_id %s published to topic %s", c.ID, *topic))

	tk := &protomfx.ThingKey{
		Value: string(c.Password),
		Type:  c.Username,
	}

	pc, err := h.things.GetPubConfigByKey(context.Background(), tk)
	if err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
		return
	}

	subject, subtopic, err := parseTopic(*topic, pc.GetPublisherID())
	if err != nil {
		h.logger.Error(errors.Wrap(errFailedParseSubtopic, err).Error())
		return
	}

	msg := protomfx.Message{
		Protocol: protocol,
		Subtopic: subtopic,
		Payload:  *payload,
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
		return
	}

	if err := h.publisher.Publish(subject, msg); err != nil {
		h.logger.Error(errors.Wrap(messaging.ErrPublishMessage, err).Error())
	}
}

// parseTopic parses an MQTT topic and returns the NATS subject and
// the message subtopic (the trailing path after prefix/id/suffix).
// Commands topics are routed to their own NATS subjects; everything else is
// routed to the publisher's messages subject.
func parseTopic(topic, publisherID string) (subject, subtopic string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) >= 3 && parts[1] != "" {
		prefix, id, suffix := parts[0], parts[1], parts[2]
		rest := ""
		if len(parts) > 3 {
			if rest, err = messaging.NormalizeSubtopic(strings.Join(parts[3:], "/")); err != nil {
				return "", "", err
			}
		}
		switch {
		case prefix == topicPrefixThings && suffix == topicSuffixCommands:
			return nats.GetThingCommandsSubject(id, rest), rest, nil
		case prefix == topicPrefixGroups && suffix == topicSuffixCommands:
			return nats.GetGroupCommandsSubject(id, rest), rest, nil
		// Route to the topic's target thing subject, not the publisher's.
		case prefix == topicPrefixThings && suffix == topicSuffixMessages:
			return nats.GetMessagesSubject(id, rest), rest, nil
		}
	}

	// Default: full normalized topic as subtopic, routed to publisher's messages subject.
	normalizedTopic, err := messaging.NormalizeSubtopic(topic)
	if err != nil {
		return "", "", err
	}
	return nats.GetMessagesSubject(publisherID, normalizedTopic), normalizedTopic, nil
}

// Subscribe - after client successfully subscribed
func (h *handler) Subscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error(errors.Wrap(errFailedSubscribe, ErrClientNotInitialized).Error())
		return
	}

	subs, err := h.getSubscriptions(c, topics)
	if err != nil {
		h.logger.Error(errors.Wrap(errFailedSubscribe, err).Error())
		return
	}

	for _, s := range subs {
		if err = h.service.CreateSubscription(context.Background(), s); err != nil {
			h.logger.Error(errors.Wrap(errFailedSubscribe, err).Error())
			return
		}
	}

	h.logger.Info(fmt.Sprintf("client_id %s subscribed to topics %s", c.ID, strings.Join(*topics, ", ")))
}

// Unsubscribe - after client unsubscribed
func (h *handler) Unsubscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error(errors.Wrap(errFailedUnsubscribe, ErrClientNotInitialized).Error())
		return
	}

	subs, err := h.getSubscriptions(c, topics)
	if err != nil {
		h.logger.Error(errors.Wrap(errFailedUnsubscribe, err).Error())
		return
	}

	for _, s := range subs {
		if err = h.service.RemoveSubscription(context.Background(), s); err != nil {
			h.logger.Error(errors.Wrap(errFailedUnsubscribe, err).Error())
			return
		}
	}

	h.logger.Info(fmt.Sprintf("client_id %s unsubscribed from topics %s", c.ID, strings.Join(*topics, ", ")))
}

// Disconnect - connection with broker or client lost
func (h *handler) Disconnect(c *session.Client) {
	if c == nil {
		h.logger.Error(errors.Wrap(errFailedDisconnect, ErrClientNotInitialized).Error())
		return
	}

	if err := h.cache.Disconnect(context.Background(), c.ID); err != nil {
		h.logger.Error(errors.Wrap(errFailedCacheDisconnection, err).Error())
	}

	h.logger.Info(fmt.Sprintf("client_id %s disconnected", c.ID))
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
		h.logger.Error(errors.Wrap(errFailedCacheConnection, err).Error())
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
	// Reject leading-slash variants of the custom topic patterns.
	if strings.HasPrefix(topic, "/"+topicPrefixThings+"/") || strings.HasPrefix(topic, "/"+topicPrefixGroups+"/") {
		return errors.Wrap(ErrUnauthorizedSubscriptionTopic, fmt.Errorf("%s (leading slash not allowed)", topic))
	}

	if !strings.HasPrefix(topic, topicPrefixThings+"/") && !strings.HasPrefix(topic, topicPrefixGroups+"/") {
		return nil
	}

	parts := strings.Split(topic, "/")
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
