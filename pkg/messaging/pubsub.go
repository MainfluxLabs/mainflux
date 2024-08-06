// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	SenMLContentType = "application/senml+json"
	CBORContentType  = "application/senml+cbor"
	JSONContentType  = "application/json"
	SenMLFormat      = "senml"
	JSONFormat       = "json"
	CBORFormat       = "cbor"
	regExParts       = 2
)

var subtopicRegExp = regexp.MustCompile(`(?:^/channels/[\w\-]+)?/messages(/[^?]*)?(\?.*)?$`)

var (
	// ErrConnect indicates that connection to MQTT broker failed
	ErrConnect = errors.New("failed to connect to MQTT broker")

	// ErrPublishTimeout indicates that the publishing failed due to timeout.
	ErrPublishTimeout = errors.New("failed to publish due to timeout reached")

	// ErrSubscribeTimeout indicates that the subscription failed due to timeout.
	ErrSubscribeTimeout = errors.New("failed to subscribe due to timeout reached")

	// ErrUnsubscribeTimeout indicates that unsubscribe failed due to timeout.
	ErrUnsubscribeTimeout = errors.New("failed to unsubscribe due to timeout reached")

	// ErrUnsubscribeDeleteTopic indicates that unsubscribe failed because the topic was deleted.
	ErrUnsubscribeDeleteTopic = errors.New("failed to unsubscribe due to deletion of topic")

	// ErrNotSubscribed indicates that the topic is not subscribed to.
	ErrNotSubscribed = errors.New("not subscribed")

	// ErrEmptyTopic indicates the absence of topic.
	ErrEmptyTopic = errors.New("empty topic")

	// ErrMalformedSubtopic indicates that the subtopic is malformed.
	ErrMalformedSubtopic = errors.New("malformed subtopic")

	// ErrEmptyID indicates the absence of ID.
	ErrEmptyID = errors.New("empty ID")
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publish publishes message to the message broker.
	Publish(msg protomfx.Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// MessageHandler represents protomfx.Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(msg protomfx.Message) error

	// Cancel is used for cleanup during unsubscribing and it's optional.
	Cancel() error
}

// Subscriber specifies message subscription API.
type Subscriber interface {
	// Subscribe subscribes to the message stream and consumes messages.
	Subscribe(id, topic string, handler MessageHandler) error

	// Unsubscribe unsubscribes from the message stream and
	// stops consuming messages.
	Unsubscribe(id, topic string) error

	// Close gracefully closes message subscriber's connection.
	Close() error
}

// PubSub  represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}

func CreateMessage(conn *protomfx.ConnByKeyRes, protocol, subject string, payload *[]byte) protomfx.Message {
	msg := protomfx.Message{
		Protocol:  protocol,
		Channel:   conn.ChannelID,
		Subtopic:  subject,
		Publisher: conn.ThingID,
		Payload:   *payload,
		Created:   time.Now().UnixNano(),
		Profile:   &protomfx.Profile{},
	}

	if conn.Profile == nil {
		return msg
	}

	msg.Profile.Write = conn.Profile.Write
	msg.Profile.WebhookID = conn.Profile.WebhookID
	msg.Profile.SmtpID = conn.Profile.SmtpID
	msg.Profile.SmppID = conn.Profile.SmppID
	msg.Profile.ContentType = conn.Profile.ContentType

	if conn.Profile.Transformer != nil {
		msg.Profile.Transformer = &protomfx.Transformer{
			ValueFields:  conn.Profile.Transformer.ValueFields,
			TimeField:    conn.Profile.Transformer.TimeField,
			TimeFormat:   conn.Profile.Transformer.TimeFormat,
			TimeLocation: conn.Profile.Transformer.TimeLocation,
		}
	}

	return msg
}

func ExtractSubtopic(path string) (string, error) {
	subtopicParts := subtopicRegExp.FindStringSubmatch(path)
	if len(subtopicParts) < regExParts {
		return "", ErrMalformedSubtopic
	}

	return subtopicParts[1], nil
}

func CreateSubject(subtopic string) (string, error) {
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
