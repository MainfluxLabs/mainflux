// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
)

const sep = "/"

var (
	keys = [...]string{"publisher", "protocol", "channel", "subtopic"}

	// ErrTransform represents an error during parsing message.
	ErrTransform = errors.New("unable to parse JSON object")
	// ErrFormatPayload represents an error during formatting message payload.
	ErrFormatPayload = errors.New("unable to format payload")
	// ErrInvalidKey represents the use of a reserved message field.
	ErrInvalidKey = errors.New("invalid object key")
	// ErrInvalidTimeField represents the use an invalid time field.
	ErrInvalidTimeField = errors.New("invalid time field")

	errUnknownFormat     = errors.New("unknown format of JSON message")
	errInvalidFormat     = errors.New("invalid JSON object")
	errInvalidNestedJSON = errors.New("invalid nested JSON object")
)

// TimeField represents the message fields to use as timestamp
type TimeField struct {
	Name     string `json:"name"`
	Format   string `json:"format"`
	Location string `json:"location"`
}

type transformerService struct {
	timeFields []TimeField
}

// New returns a new JSON transformer.
func New() transformers.Transformer {
	return &transformerService{}
}

// Transform transforms Mainflux message to a list of JSON messages.
func (ts *transformerService) Transform(msg protomfx.Message) (interface{}, error) {
	ret := Message{
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Protocol:  msg.Protocol,
	}

	if msg.Profile.WebhookID != "" {
		ret.Profile = map[string]interface{}{
			"webhook_id": msg.Profile.WebhookID,
		}
	}

	subs := strings.Split(msg.Subtopic, ".")
	switch {
	case msg.Subtopic == "":
		return nil, errors.Wrap(ErrTransform, errUnknownFormat)
	case len(subs) == 1:
		ret.Subtopic = ""
	default:
		ret.Subtopic = strings.Join(subs[:len(subs)-1], ".")
	}

	format := subs[len(subs)-1]
	var payload interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, errors.Wrap(ErrTransform, err)
	}

	switch p := payload.(type) {
	case map[string]interface{}:
		formattedPayload, err := transformPayload(p, msg.Profile.Transformer.ValueFields)
		if err != nil {
			return nil, errors.Wrap(ErrFormatPayload, err)
		}
		ret.Payload = formattedPayload

		// Apply timestamp transformation rules depending on key/unit pairs
		ts, err := ts.transformTimeField(p, *msg.Profile.Transformer)
		if err != nil {
			return nil, errors.Wrap(ErrInvalidTimeField, err)
		}
		if ts != 0 {
			ret.Created = ts
		}

		return Messages{[]Message{ret}, format}, nil
	case []interface{}:
		res := []Message{}
		// Make an array of messages from the root array.
		for _, val := range p {
			v, ok := val.(map[string]interface{})
			if !ok {
				return nil, errors.Wrap(ErrTransform, errInvalidNestedJSON)
			}
			newMsg := ret

			formattedPayload, err := transformPayload(v, msg.Profile.Transformer.ValueFields)
			if err != nil {
				return nil, errors.Wrap(ErrFormatPayload, err)
			}
			newMsg.Payload = formattedPayload

			// Apply timestamp transformation rules depending on key/unit pairs
			ts, err := ts.transformTimeField(v, *msg.Profile.Transformer)
			if err != nil {
				return nil, errors.Wrap(ErrInvalidTimeField, err)
			}
			if ts != 0 {
				ret.Created = ts
			}

			res = append(res, newMsg)
		}
		return Messages{res, format}, nil
	default:
		return nil, errors.Wrap(ErrTransform, errInvalidFormat)
	}
}

// ParseFlat receives flat map that represents complex JSON objects and returns
// the corresponding complex JSON object with nested maps. It's the opposite
// of the Flatten function.
func ParseFlat(flat interface{}) interface{} {
	msg := make(map[string]interface{})
	switch v := flat.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if value == nil {
				continue
			}
			subKeys := strings.Split(key, sep)
			n := len(subKeys)
			if n == 1 {
				msg[key] = value
				continue
			}
			current := msg
			for i, k := range subKeys {
				if _, ok := current[k]; !ok {
					current[k] = make(map[string]interface{})
				}
				if i == n-1 {
					current[k] = value
					break
				}
				current = current[k].(map[string]interface{})
			}
		}
	}
	return msg
}

// Flatten makes nested maps flat using composite keys created by concatenation of the nested keys.
func Flatten(m map[string]interface{}) (map[string]interface{}, error) {
	return flatten("", make(map[string]interface{}), m)
}

func flatten(prefix string, m, m1 map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range m1 {
		if strings.Contains(k, sep) {
			return nil, ErrInvalidKey
		}
		for _, key := range keys {
			if k == key {
				return nil, ErrInvalidKey
			}
		}
		switch val := v.(type) {
		case map[string]interface{}:
			var err error
			m, err = flatten(prefix+k+sep, m, val)
			if err != nil {
				return nil, err
			}
		default:
			m[prefix+k] = v
		}
	}
	return m, nil
}

func (ts *transformerService) transformTimeField(payload map[string]interface{}, transformer protomfx.Transformer) (int64, error) {
	if transformer.TimeField == "" {
		return 0, nil
	}

	if val, ok := payload[transformer.TimeField]; ok {
		t, err := parseTimestamp(transformer.TimeFormat, val, transformer.TimeLocation)
		if err != nil {
			return 0, err
		}
		return t.UnixNano(), nil
	}

	return 0, nil
}

func transformPayload(payload map[string]interface{}, fieldValues []string) (map[string]interface{}, error) {
	formattedPayload := make(map[string]interface{})

	for _, fv := range fieldValues {
		if value, ok := payload[fv]; ok {
			formattedPayload[fv] = value
		}
	}

	return formattedPayload, nil
}
