// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

const (
	SenMLContentType = "application/senml+json"
	JSONContentType  = "application/json"
)

// ErrInvalidContentType indicates an invalid Content-Type
var ErrInvalidContentType = errors.New("invalid content type")

func FormatMessage(pc domain.PubConfigInfo, msg *protomfx.Message) error {
	msg.ThingId = pc.ThingID
	msg.Created = time.Now().UnixNano()

	if pc.ProfileConfig != nil {
		msg.ContentType = pc.ProfileConfig.ContentType
		switch msg.ContentType {
		case JSONContentType:
			if err := mfjson.TransformPayload(pc.ProfileConfig.Transformer, msg); err != nil {
				return err
			}
		case SenMLContentType:
			if err := senml.TransformPayload(msg); err != nil {
				return err
			}
		default:
			return ErrInvalidContentType
		}
	}

	return nil
}

func PayloadHash(payload []byte) int32 {
	h := fnv.New32a()
	h.Write(payload)
	return int32(h.Sum32())
}

func ToJSONMessage(message protomfx.Message) mfjson.Message {
	return mfjson.Message{
		Created:  message.Created,
		Subtopic: message.Subtopic,
		ThingID:  message.ThingId,
		Protocol: message.Protocol,
		Payload:  message.Payload,
	}
}

// ToJSONMessages converts each record of message into a JSON message.
// A message without records is converted as a single JSON message.
func ToJSONMessages(message protomfx.Message) []mfjson.Message {
	if len(message.Records) == 0 {
		return []mfjson.Message{ToJSONMessage(message)}
	}

	var msgs []mfjson.Message
	for _, record := range message.Records {
		msgs = append(msgs, mfjson.Message{
			Created:  record.Created,
			Subtopic: message.Subtopic,
			ThingID:  message.ThingId,
			Protocol: message.Protocol,
			Payload:  record.Payload,
		})
	}

	return msgs
}

func ToJSONCommand(cmd protomfx.Command) mfjson.Command {
	return mfjson.Command{
		Created:     cmd.Created,
		Subtopic:    cmd.Subtopic,
		ThingID:     cmd.ThingId,
		RecipientID: cmd.RecipientId,
		Protocol:    cmd.Protocol,
		Payload:     cmd.Payload,
	}
}

func ToSenMLMessage(message protomfx.Message) (senml.Message, error) {
	var msg senml.Message
	if err := json.Unmarshal(message.Payload, &msg); err != nil {
		return senml.Message{}, err
	}

	msg.ThingID = message.ThingId
	msg.Subtopic = message.Subtopic
	msg.Protocol = message.Protocol

	return msg, nil
}

func SplitMessage(message protomfx.Message) ([]protomfx.Message, error) {
	var payload any
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, err
	}

	if pyds, ok := payload.([]any); ok {
		var messages []protomfx.Message
		for _, pyd := range pyds {
			data, err := json.Marshal(pyd)
			if err != nil {
				return nil, err
			}
			newMsg := message
			newMsg.Payload = data
			messages = append(messages, newMsg)
		}
		return messages, nil
	}

	return []protomfx.Message{message}, nil
}
