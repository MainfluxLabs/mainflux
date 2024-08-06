// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/mqtt/redis"
	"github.com/MainfluxLabs/mainflux/pkg/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const (
	protocol  = "mqtt"
	connected = "connected"
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

	if _, err := h.authAccess(c); err != nil {
		return err
	}

	return nil
}

// AuthSubscribe is called on device subscribe,
// prior creating a subscription on the MQTT broker
func (h *handler) AuthSubscribe(c *session.Client, topics *[]string) error {
	if c == nil {
		return ErrClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return ErrMissingTopicSub
	}

	if _, err := h.authAccess(c); err != nil {
		return err
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

	subtopic, err := messaging.ExtractSubtopic(*topic)
	if err != nil {
		h.logger.Error(LogErrFailedPublish + (ErrMalformedTopic).Error())
		return
	}

	subject, err := messaging.CreateSubject(subtopic)
	if err != nil {
		h.logger.Error(logErrFailedParseSubtopic + err.Error())
		return
	}

	conn, err := h.auth.GetConnByKey(context.Background(), string(c.Password))
	if err != nil {
		h.logger.Error(LogErrFailedPublish + (ErrAuthentication).Error())
	}

	m := messaging.CreateMessage(&conn, protocol, subject, payload)

	for _, pub := range h.publishers {
		if err := pub.Publish(m); err != nil {
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

func (h *handler) authAccess(c *session.Client) (protomfx.ConnByKeyRes, error) {
	conn, err := h.auth.GetConnByKey(context.Background(), string(c.Password))
	if err != nil {
		return protomfx.ConnByKeyRes{}, err
	}

	if conn.ThingID != c.Username {
		return protomfx.ConnByKeyRes{}, ErrAuthentication
	}

	return conn, nil
}

func (h *handler) getSubcriptions(c *session.Client, topics *[]string) ([]Subscription, error) {
	var subs []Subscription
	for _, t := range *topics {

		subtopic, err := messaging.ExtractSubtopic(t)
		if err != nil {
			return nil, err
		}

		conn, err := h.auth.GetConnByKey(context.Background(), string(c.Password))
		if err != nil {
			return nil, err
		}

		subject, err := messaging.CreateSubject(subtopic)
		if err != nil {
			return nil, err
		}

		sub := Subscription{
			Subtopic:  subject,
			ChanID:    conn.ChannelID,
			ThingID:   c.Username,
			ClientID:  c.ID,
			Status:    connected,
			CreatedAt: float64(time.Now().UnixNano()) / float64(1e9),
		}
		subs = append(subs, sub)
	}

	return subs, nil
}
