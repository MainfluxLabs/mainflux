// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	broker "github.com/nats-io/nats.go"
)

const (
	// SubjectThings represents the wildcard subject covering all thing subjects.
	SubjectThings = "things.>"
	// SubjectGroups represents the wildcard subject covering all group subjects.
	SubjectGroups = "groups.>"
	// SubjectMessages represents subject used to subscribe to the global message stream.
	SubjectMessages = "things.*.messages"
	// SubjectMessagesWithSubtopic represents subject used to subscribe to the global message stream with subtopic.
	SubjectMessagesWithSubtopic = "things.*.messages.>"
	// SubjectThingCommands represents subject used to subscribe to thing commands.
	SubjectThingCommands = "things.*.commands"
	// SubjectThingCommandsWithSubtopic represents subject used to subscribe to thing commands with subtopic.
	SubjectThingCommandsWithSubtopic = "things.*.commands.>"
	// SubjectGroupCommands represents subject used to subscribe to group commands.
	SubjectGroupCommands = "groups.*.commands"
	// SubjectGroupCommandsWithSubtopic represents subject used to subscribe to group commands with subtopic.
	SubjectGroupCommandsWithSubtopic = "groups.*.commands.>"
	// SubjectSmtp represents subject used to subscribe to SMTP notifications.
	SubjectSmtp = "smtp.*"
	// SubjectSmpp represents subject used to subscribe to SMPP notifications.
	SubjectSmpp = "smpp.*"
	// SubjectAlarms represents subject used to subscribe to alarm triggers.
	SubjectAlarms = "alarms.*"
	// SubjectWebhooks represents subject used to subscribe to webhook forwarding — both rule-triggered and integration-based.
	SubjectWebhooks = "webhooks"
	// SubjectRules represents subject used to route messages to the rules service.
	SubjectRules = "rules"
)

type subscription struct {
	*broker.Subscription
	cancel func() error
}

type pubsub struct {
	publisher
	logger        log.Logger
	mu            sync.Mutex
	queue         string
	subscriptions map[string]map[string]subscription
}

// NewPubSub returns NATS message publisher/subscriber.
// Parameter queue specifies the queue for the Subscribe method.
// If queue is specified (is not an empty string), Subscribe method
// will execute NATS QueueSubscribe which is conceptually different
// from ordinary subscribe. For more information, please take a look
// here: https://docs.nats.io/developing-with-nats/receiving/queues.
// If the queue is empty, Subscribe will be used.
func NewPubSub(url, queue string, logger log.Logger) (*pubsub, error) {
	conn, js, err := connect(url)
	if err != nil {
		return nil, err
	}

	ret := &pubsub{
		publisher: publisher{
			conn: conn,
			js:   js,
		},
		queue:         queue,
		logger:        logger,
		subscriptions: make(map[string]map[string]subscription),
	}

	return ret, nil
}

func (ps *pubsub) Subscribe(id, topic string, handler messaging.MessageHandler) error {
	newFn := func() proto.Message { return &protomfx.Message{} }
	handleFn := func(subject string, msg proto.Message) error {
		v, ok := msg.(*protomfx.Message)
		if !ok {
			return fmt.Errorf("nats: unexpected message type %T for message subject", msg)
		}
		return handler.Handle(subject, *v)
	}
	return ps.subscribe(id, topic, newFn, handleFn, handler.Cancel)
}

func (ps *pubsub) Unsubscribe(id, topic string) error {
	if id == "" {
		return messaging.ErrEmptyID
	}
	if topic == "" {
		return messaging.ErrEmptyTopic
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.unsubscribe(id, topic)
}

func (ps *pubsub) SubscribeAlarms(id string, handler messaging.AlarmHandler) error {
	newFn := func() proto.Message { return &protomfx.Alarm{} }
	handleFn := func(subject string, msg proto.Message) error {
		v, ok := msg.(*protomfx.Alarm)
		if !ok {
			return fmt.Errorf("nats: unexpected message type %T for alarm subject", msg)
		}
		return handler(subject, *v)
	}
	return ps.subscribe(id, SubjectAlarms, newFn, handleFn, nil)
}

func (ps *pubsub) SubscribeCommands(id, topic string, handler messaging.CommandHandler) error {
	newFn := func() proto.Message { return &protomfx.Command{} }
	handleFn := func(subject string, msg proto.Message) error {
		v, ok := msg.(*protomfx.Command)
		if !ok {
			return fmt.Errorf("nats: unexpected message type %T for command subject", msg)
		}
		return handler(subject, *v)
	}
	return ps.subscribe(id, topic, newFn, handleFn, nil)
}

func (ps *pubsub) SubscribeNotifications(id, topic string, handler messaging.NotificationHandler) error {
	newFn := func() proto.Message { return &protomfx.Notification{} }
	handleFn := func(subject string, msg proto.Message) error {
		v, ok := msg.(*protomfx.Notification)
		if !ok {
			return fmt.Errorf("nats: unexpected message type %T for notification subject", msg)
		}
		return handler(subject, *v)
	}
	return ps.subscribe(id, topic, newFn, handleFn, nil)
}

func (ps *pubsub) SubscribeWebhooks(id string, handler messaging.WebhookHandler) error {
	newFn := func() proto.Message { return &protomfx.Webhook{} }
	handleFn := func(subject string, msg proto.Message) error {
		v, ok := msg.(*protomfx.Webhook)
		if !ok {
			return fmt.Errorf("nats: unexpected message type %T for webhook subject", msg)
		}
		return handler(subject, *v)
	}
	return ps.subscribe(id, SubjectWebhooks, newFn, handleFn, nil)
}

// unsubscribe removes the subscription for the given id and topic.
// Must be called with ps.mu held.
func (ps *pubsub) unsubscribe(id, topic string) error {
	s, ok := ps.subscriptions[topic]
	if !ok {
		return messaging.ErrNotSubscribed
	}
	current, ok := s[id]
	if !ok {
		return messaging.ErrNotSubscribed
	}
	if current.cancel != nil {
		if err := current.cancel(); err != nil {
			return err
		}
	}
	if err := current.Unsubscribe(); err != nil {
		return err
	}
	delete(s, id)
	if len(s) == 0 {
		delete(ps.subscriptions, topic)
	}
	return nil
}

// subscribe is the shared Subscribe entry point for every typed stream.
// newFn allocates the concrete proto.Message to unmarshal into, and handleFn
// dispatches the unmarshaled message to the caller's typed handler.
func (ps *pubsub) subscribe(id, topic string, newFn func() proto.Message, handleFn func(subject string, msg proto.Message) error, cancelFn func() error) error {
	if id == "" {
		return messaging.ErrEmptyID
	}
	if topic == "" {
		return messaging.ErrEmptyTopic
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	if s, ok := ps.subscriptions[topic]; ok {
		if _, ok := s[id]; ok {
			if err := ps.unsubscribe(id, topic); err != nil {
				return err
			}
		}
	}

	s, ok := ps.subscriptions[topic]
	if !ok {
		s = make(map[string]subscription)
		ps.subscriptions[topic] = s
	}

	nh := ps.natsGenericHandler(newFn, handleFn)
	var (
		sub *broker.Subscription
		err error
	)
	durable := durableName(ps.queue, id, topic)
	switch ps.queue {
	case "":
		sub, err = ps.js.Subscribe(topic, nh, broker.Durable(durable), broker.DeliverAll())
	default:
		sub, err = ps.js.QueueSubscribe(topic, ps.queue, nh, broker.Durable(durable), broker.DeliverAll())
	}
	if err != nil {
		return err
	}

	s[id] = subscription{Subscription: sub, cancel: cancelFn}
	return nil
}

// natsGenericHandler wraps newFn/handleFn into the broker's raw message callback:
// it allocates the concrete proto.Message, unmarshals into it, then dispatches.
func (ps *pubsub) natsGenericHandler(newFn func() proto.Message, handleFn func(subject string, msg proto.Message) error) broker.MsgHandler {
	return func(m *broker.Msg) {
		msg := newFn()
		if err := proto.Unmarshal(m.Data, msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received %T: %s", msg, err))
			return
		}
		if err := handleFn(m.Subject, msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle %T: %s", msg, err))
		}
	}
}
