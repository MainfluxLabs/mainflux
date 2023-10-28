// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/mqtt/redis"
	"github.com/MainfluxLabs/mainflux/pkg/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const (
	protocol     = "mqtt"
	connected    = "connected"
	disconnected = "disconnected"
)

const (
	LogInfoSubscribed                  = "subscribed with client_id %s to topics %s"
	LogInfoUnsubscribed                = "unsubscribed client_id %s from topics %s"
	LogInfoConnected                   = "connected with client_id %s"
	LogInfoDisconnected                = "disconnected client_id %s and username %s"
	LogInfoPublished                   = "published with client_id %s to the topic %s"
	LogErrFailedConnect                = "failed to connect: "
	LogErrFailedSubscribe              = "failed to subscribe: "
	LogErrFailedUnsubscribe            = "failed to unsubscribe: "
	LogErrFailedPublish                = "failed to publish: "
	LogErrFailedDisconnect             = "failed to disconnect: "
	LogErrFailedPublishDisconnectEvent = "failed to publish disconnect event: "
	logErrFailedParseSubtopic          = "failed to parse subtopic: "
	LogErrFailedPublishConnectEvent    = "failed to publish connect event: "
	LogErrFailedPublishToMsgBroker     = "failed to publish to mainflux message broker: "
)

var (
	channelRegExp                = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	ErrMalformedSubtopic         = errors.New("malformed subtopic")
	ErrClientNotInitialized      = errors.New("client is not initialized")
	ErrMalformedTopic            = errors.New("malformed topic")
	ErrMissingClientID           = errors.New("client_id not found")
	ErrMissingTopicPub           = errors.New("failed to publish due to missing topic")
	ErrMissingTopicSub           = errors.New("failed to subscribe due to missing topic")
	ErrAuthentication            = errors.New("failed to perform authentication over the entity")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
)

// Event implements events.Event interface
type handler struct {
	publishers []messaging.Publisher
	auth       auth.Client
	logger     logger.Logger
	es         redis.EventStore
	service    Service
}

// NewHandler creates new Handler entity
func NewHandler(publishers []messaging.Publisher, es redis.EventStore,
	logger logger.Logger, auth auth.Client, svc Service) session.Handler {
	return &handler{
		es:         es,
		logger:     logger,
		publishers: publishers,
		auth:       auth,
		service:    svc,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the MQTT broker
func (h *handler) AuthConnect(c *session.Client) error {
	if c == nil {
		return ErrClientNotInitialized
	}

	if c.ID == "" {
		return ErrMissingClientID
	}

	thid, err := h.auth.Identify(context.Background(), string(c.Password))
	if err != nil {
		return err
	}

	if thid != c.Username {
		return errors.ErrAuthentication
	}

	if err := h.es.Connect(c.Username); err != nil {
		h.logger.Error(LogErrFailedPublishConnectEvent + err.Error())
	}

	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthPublish(c *session.Client, topic *string, payload *[]byte) error {
	if c == nil {
		return ErrClientNotInitialized
	}
	if topic == nil {
		return ErrMissingTopicPub
	}

	if err := h.authAccess(c, *topic); err != nil {
		return err
	}

	return nil
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthSubscribe(c *session.Client, topics *[]string) error {
	if c == nil {
		return ErrClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return ErrMissingTopicSub
	}

	for _, t := range *topics {
		err := h.authAccess(c, t)
		if err != nil {
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
		h.logger.Error(LogErrFailedPublish + ErrClientNotInitialized.Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, c.ID, *topic))
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 2 {
		h.logger.Error(LogErrFailedPublish + (ErrMalformedTopic).Error())
		return
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		h.logger.Error(logErrFailedParseSubtopic + err.Error())
		return
	}

	msg := messaging.Message{
		Protocol:  protocol,
		Channel:   chanID,
		Subtopic:  subtopic,
		Publisher: c.Username,
		Payload:   *payload,
		Created:   time.Now().UnixNano(),
	}

	for _, pub := range h.publishers {
		if err := pub.Publish(msg.Channel, msg); err != nil {
			h.logger.Error(LogErrFailedPublishToMsgBroker + err.Error())
		}
	}
}

// Subscribe - after client successfully subscribed
func (h *handler) Subscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error(LogErrFailedSubscribe + (ErrClientNotInitialized).Error())
		return
	}

	subs, err := h.getSubcriptions(c, topics)
	if err != nil {
		h.logger.Error(LogErrFailedSubscribe + err.Error())
		return
	}

	for _, s := range subs {
		err = h.service.CreateSubscription(context.Background(), s)
		if err != nil {
			h.logger.Error(LogErrFailedSubscribe + (ErrSubscriptionAlreadyExists).Error())
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

	subs, err := h.getSubcriptions(c, topics)
	if err != nil {
		h.logger.Error(LogErrFailedSubscribe + err.Error())
		return
	}

	for _, s := range subs {
		if h.service.RemoveSubscription(context.Background(), s); err != nil {
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

	h.logger.Error(fmt.Sprintf(LogInfoDisconnected, c.ID, c.Username))
	if err := h.es.Disconnect(c.Username); err != nil {
		h.logger.Error(LogErrFailedPublishDisconnectEvent + err.Error())
	}
}

func (h *handler) authAccess(c *session.Client, topic string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.Match([]byte(topic)) {
		return ErrMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return ErrMalformedTopic
	}

	thID, _, err := h.auth.ConnectionByThingKey(context.Background(), string(c.Password))
	if err != nil {
		return err
	}

	if thID != c.Username {
		return ErrAuthentication
	}

	return nil
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", ErrMalformedSubtopic
	}
	subtopic = strings.Replace(subtopic, "/", ".", -1)

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", ErrMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}

func (h *handler) getSubcriptions(c *session.Client, topics *[]string) ([]Subscription, error) {
	var subs []Subscription
	for _, t := range *topics {
		channelAndSubtopic := channelRegExp.FindStringSubmatch(t)
		if len(channelAndSubtopic) < 2 {
			return nil, ErrMalformedTopic
		}

		chanID := channelAndSubtopic[1]
		subtopic := channelAndSubtopic[2]

		subtopic, err := parseSubtopic(subtopic)
		if err != nil {
			return nil, err
		}

		sub := Subscription{
			Subtopic:  subtopic,
			ChanID:    chanID,
			ThingID:   c.Username,
			ClientID:  c.ID,
			Status:    connected,
			CreatedAt: float64(time.Now().UnixNano()) / float64(1e9),
		}
		subs = append(subs, sub)
	}

	return subs, nil
}
