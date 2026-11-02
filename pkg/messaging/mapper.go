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

// FormatMessage assigns the thing identifier and the timestamp to msg
// and transforms its records according to the profile configuration.
func FormatMessage(pc domain.PubConfigInfo, msg *protomfx.Message) error {
	msg.ThingId = pc.ThingID
	msg.Created = time.Now().UnixNano()
	for _, record := range msg.Records {
		if record.Time == 0 {
			record.Time = msg.Created
		}
	}

	if pc.ProfileConfig == nil {
		return nil
	}

	msg.ContentType = pc.ProfileConfig.ContentType
	var records []*protomfx.Record
	for _, record := range msg.Records {
		transformed, err := transformRecord(pc.ProfileConfig, record)
		if err != nil {
			return err
		}
		records = append(records, transformed...)
	}
	msg.Records = records

	// A single JSON record carries the device time of the message.
	if msg.ContentType == JSONContentType && len(msg.Records) == 1 {
		msg.Created = msg.Records[0].Time
	}

	return nil
}

// transformRecord converts a record into the records of the profile content type.
func transformRecord(config *domain.ProfileConfig, record *protomfx.Record) ([]*protomfx.Record, error) {
	switch config.ContentType {
	case JSONContentType:
		return mfjson.TransformPayload(config.Transformer, record.Payload, record.Time)
	case SenMLContentType:
		return senml.TransformPayload(senml.JSON, record.Payload, record.Time)
	default:
		return nil, ErrInvalidContentType
	}
}

// ToPayload returns the payload of a single non-SenML record unchanged
// and encodes the payloads of all other records as a JSON array.
func ToPayload(message protomfx.Message) ([]byte, error) {
	if message.ContentType != SenMLContentType && len(message.Records) == 1 {
		return message.Records[0].Payload, nil
	}

	payloads := make([]json.RawMessage, 0, len(message.Records))
	for _, record := range message.Records {
		payloads = append(payloads, record.Payload)
	}

	return json.Marshal(payloads)
}

func PayloadHash(payload []byte) int32 {
	h := fnv.New32a()
	h.Write(payload)
	return int32(h.Sum32())
}

func ToJSONMessage(message protomfx.Message) (mfjson.Message, error) {
	payload, err := ToPayload(message)
	if err != nil {
		return mfjson.Message{}, err
	}

	return mfjson.Message{
		Created:  message.Created,
		Subtopic: message.Subtopic,
		ThingID:  message.ThingId,
		Protocol: message.Protocol,
		Payload:  payload,
	}, nil
}

// ToJSONMessages converts each record of message into a JSON message.
func ToJSONMessages(message protomfx.Message) []mfjson.Message {
	var msgs []mfjson.Message
	for _, record := range message.Records {
		msgs = append(msgs, mfjson.Message{
			Created:  record.Time,
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

// ToSenMLMessages converts each record of message into a SenML message.
func ToSenMLMessages(message protomfx.Message) ([]senml.Message, error) {
	var msgs []senml.Message
	for _, record := range message.Records {
		var msg senml.Message
		if err := json.Unmarshal(record.Payload, &msg); err != nil {
			return nil, err
		}

		msg.ThingID = message.ThingId
		msg.Subtopic = message.Subtopic
		msg.Protocol = message.Protocol
		msgs = append(msgs, msg)
	}

	return msgs, nil
}
