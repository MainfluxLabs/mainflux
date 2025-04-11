// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"
	"net/url"
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

func CreateSubject(topic string) (string, error) {
	// Handle cases where full path might be passed (backward compatibility)
	if strings.HasPrefix(topic, "/messages/") {
		topic = strings.TrimPrefix(topic, "/messages/")
	} else if topic == "/messages" {
		return "", nil
	}

	if topic == "" {
		return topic, nil
	}

	// Handle wildcards
	if topic == ">" || strings.Contains(topic, "*") {
		return topic, nil
	}

	// URL decode if needed
	decoded, err := url.QueryUnescape(topic)
	if err != nil {
		return "", ErrMalformedSubtopic
	}

	// Replace slashes with dots
	normalized := strings.Replace(decoded, "/", ".", -1)

	// Split and filter empty elements
	elems := strings.Split(normalized, ".")
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

	return strings.Join(filteredElems, "."), nil
}

func FormatMessage(pc *protomfx.PubConfByKeyRes, msg *protomfx.Message) {
	msg.Publisher = pc.PublisherID
	msg.Created = time.Now().UnixNano()

	if pc.ProfileConfig != nil {
		msg.ContentType = pc.ProfileConfig.ContentType
		msg.WriteEnabled = pc.ProfileConfig.Write
		msg.Transformer = pc.ProfileConfig.Transformer
		msg.Rules = pc.ProfileConfig.Rules
	}
}

func FindParam(payload map[string]interface{}, param string) interface{} {
	for key, value := range payload {
		if key == param {
			return value
		}

		if data, ok := value.(map[string]interface{}); ok {
			if value := FindParam(data, param); value != nil {
				return value
			}
		}
	}

	return nil
}
