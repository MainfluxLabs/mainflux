// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux"
)

const (
	SenmlContentType = "application/senml+json"
	CborContentType  = "application/senml+cbor"
	JsonContentType  = "application/json"
	SenmlFormat      = "senml"
	JsonFormat       = "json"
	CborFormat       = "cbor"
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

	// ErrUnknownContent indicates that the content type is unknown.
	ErrUnknownContent = errors.New("unknown content type")
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publish publishes message to the message broker.
	Publish(profile mainflux.Profile, msg Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// MessageHandler represents Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(msg Message) error

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

func AddProfileToMessage(profile mainflux.Profile, msg Message) (Message, error) {
	if IsEmptyProfile(profile) || profile.ContentType == "" {
		msg.Profile = &Profile{
			ContentType: SenmlContentType,
			TimeField:   &TimeField{},
			Writer:      &Writer{Retain: true},
			Notifier:    &Notifier{},
		}
		return msg, nil
	}

	msg.Profile = &Profile{
		ContentType: profile.ContentType,
		TimeField:   &TimeField{},
	}

	if profile.Writer != nil {
		msg.Profile.Writer = &Writer{
			Retain:    profile.Writer.Retain,
			Subtopics: profile.Writer.Subtopics,
		}
	}

	if profile.Notifier != nil {
		msg.Profile.Notifier = &Notifier{
			Protocol:  profile.Notifier.Protocol,
			Contacts:  profile.Notifier.Contacts,
			Subtopics: profile.Notifier.Subtopics,
		}
	}

	if profile.TimeField != nil && profile.ContentType == JsonContentType {
		msg.Profile.TimeField = &TimeField{
			Name:     profile.TimeField.Name,
			Format:   profile.TimeField.Format,
			Location: profile.TimeField.Location,
		}
	}

	return msg, nil
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

func IsProfileNil(profile *mainflux.Profile) mainflux.Profile {
	if profile == nil {
		profile = &mainflux.Profile{}
	}

	return *profile
}

func IsEmptyProfile(profile mainflux.Profile) bool {
	emptyProfile := mainflux.Profile{}
	return profile.ContentType == emptyProfile.ContentType &&
		reflect.DeepEqual(profile.Writer, emptyProfile.Writer) &&
		reflect.DeepEqual(profile.Notifier, emptyProfile.Notifier) &&
		reflect.DeepEqual(profile.TimeField, emptyProfile.TimeField)
}
