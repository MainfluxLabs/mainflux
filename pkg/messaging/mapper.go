// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
)

func FormatMessage(pc domain.PubConfigInfo, msg *protomfx.Message) error {
	msg.Publisher = pc.PublisherID
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

func ToJSONMessage(message protomfx.Message) mfjson.Message {
	created, payload := extractCreated(message.Payload, message.Created)

	return mfjson.Message{
		Created:   created,
		Subtopic:  message.Subtopic,
		Publisher: message.Publisher,
		Protocol:  message.Protocol,
		Payload:   payload,
	}
}

func ToJSONCommand(cmd protomfx.Command) mfjson.Command {
	created, payload := extractCreated(cmd.Payload, cmd.Created)

	return mfjson.Command{
		Created:     created,
		Subtopic:    cmd.Subtopic,
		Publisher:   cmd.Publisher,
		RecipientID: cmd.RecipientID,
		Protocol:    cmd.Protocol,
		Payload:     payload,
	}
}

// extractCreated unwraps a "Created" field embedded in the payload itself,
// preferring it over the proto-level Created timestamp. Shared by
// ToJSONMessage and ToJSONCommand — MQTT wire delivery only ever needs to
// serialize protomfx.Message and protomfx.Command, a closed, known pair.
func extractCreated(payload []byte, created int64) (int64, []byte) {
	var p map[string]any

	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &p); err == nil {
			if payloadCreated, ok := p["Created"].(float64); ok {
				created = int64(payloadCreated)
				delete(p, "Created")

				payload, _ = json.Marshal(p)
			}
		}
	}

	return created, payload
}

func ToSenMLMessage(message protomfx.Message) (senml.Message, error) {
	var msg senml.Message
	if err := json.Unmarshal(message.Payload, &msg); err != nil {
		return senml.Message{}, err
	}

	msg.Publisher = message.Publisher
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
