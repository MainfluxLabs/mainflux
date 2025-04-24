// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrTransform represents an error during parsing message.
	ErrTransform = errors.New("unable to parse JSON object")
	// ErrInvalidTimeField represents the use an invalid time field.
	ErrInvalidTimeField = errors.New("invalid time field")

	errInvalidFormat     = errors.New("invalid JSON object")
	errInvalidNestedJSON = errors.New("invalid nested JSON object")
)

func Transform(msg protomfx.Message) ([]protomfx.Message, error) {
	var payload interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, errors.Wrap(ErrTransform, err)
	}

	extractedPayload := extractPayload(payload, msg.Transformer.DataField)

	switch p := extractedPayload.(type) {
	case map[string]interface{}:
		formattedPayload := filterPayloadFields(p, msg.Transformer.DataFilters)
		data, err := json.Marshal(formattedPayload)
		if err != nil {
			return nil, err
		}
		msg.Payload = data

		msg.Created = time.Now().UnixNano()
		// Apply timestamp transformation rules depending on key/unit pairs
		ts, err := transformTimeField(p, *msg.Transformer)
		if err != nil {
			return nil, errors.Wrap(ErrInvalidTimeField, err)
		}

		if ts != 0 {
			msg.Created = ts
		}

		return []protomfx.Message{msg}, nil

	case []interface{}:
		var msgs []protomfx.Message
		// Make an array of messages from the root array
		for _, val := range p {
			v, ok := val.(map[string]interface{})
			if !ok {
				return nil, errors.Wrap(ErrTransform, errInvalidNestedJSON)
			}

			formattedPayload := filterPayloadFields(v, msg.Transformer.DataFilters)
			data, err := json.Marshal(formattedPayload)
			if err != nil {
				return nil, err
			}

			newMsg := protomfx.Message{
				Publisher:   msg.Publisher,
				Subtopic:    msg.Subtopic,
				Payload:     data,
				ContentType: msg.ContentType,
				Protocol:    msg.Protocol,
				Created:     time.Now().UnixNano(),
			}

			// Apply timestamp transformation rules depending on key/unit pairs
			ts, err := transformTimeField(v, *msg.Transformer)
			if err != nil {
				return nil, errors.Wrap(ErrInvalidTimeField, err)
			}

			if ts != 0 {
				newMsg.Created = ts
			}

			msgs = append(msgs, newMsg)
		}

		return msgs, nil

	default:
		return nil, errors.Wrap(ErrTransform, errInvalidFormat)
	}
}

func MapMessageToJSON(msg protomfx.Message) (Message, error) {
	var payload Payload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return Message{}, err
	}

	return Message{
		Created:   msg.Created,
		Subtopic:  msg.Subtopic,
		Publisher: msg.Publisher,
		Protocol:  msg.Protocol,
		Payload:   payload,
	}, nil
}

func transformTimeField(payload interface{}, transformer protomfx.Transformer) (int64, error) {
	if transformer.TimeField == "" {
		return 0, nil
	}

	val := payload
	keys := strings.Split(transformer.TimeField, ".")
	for _, k := range keys {
		current, ok := val.(map[string]interface{})
		if !ok {
			return 0, nil
		}

		v, exists := current[k]
		if !exists {
			return 0, nil
		}
		val = v
	}

	t, err := parseTimestamp(transformer.TimeFormat, val, transformer.TimeLocation)
	if err != nil {
		return 0, err
	}
	return t.UnixNano(), nil
}

func extractPayload(payload interface{}, dataField string) interface{} {
	if dataField != "" {
		p := payload
		keys := strings.Split(dataField, ".")

		for _, k := range keys {
			if pv, ok := p.(map[string]interface{}); ok {
				if val, exists := pv[k]; exists {
					p = val
				}
			}
		}

		return p
	}

	return payload
}

func filterPayloadFields(payload map[string]interface{}, dataFilters []string) map[string]interface{} {
	if len(dataFilters) == 0 {
		return payload
	}

	filteredPayload := make(map[string]interface{})

	for _, key := range dataFilters {
		// Split nested path
		keys := strings.Split(key, ".")
		var value interface{} = payload

		// Traverse nested structure
		for _, k := range keys {
			current, ok := value.(map[string]interface{})
			if !ok {
				value = nil
				break
			}

			v, exists := current[k]
			if !exists {
				value = nil
				break
			}
			value = v
		}

		if value != nil {
			filteredPayload[key] = value
		}
	}

	return filteredPayload
}
