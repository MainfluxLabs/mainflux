// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
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

func TransformPayload(transformer domain.Transformer, msg *protomfx.Message) error {
	var payload any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return errors.Wrap(ErrTransform, err)
	}

	extractedPayload := extractPayload(payload, transformer.DataField)

	switch p := extractedPayload.(type) {
	case map[string]any:
		ts, err := transformTimeField(p, transformer)
		if err != nil {
			return errors.Wrap(ErrInvalidTimeField, err)
		}

		if ts != 0 {
			msg.Created = ts
		}

		formattedPayload := filterPayloadFields(p, transformer.DataFilters)
		data, err := json.Marshal(formattedPayload)
		if err != nil {
			return err
		}
		msg.Payload = data
		msg.Records = []*protomfx.Record{{Payload: data, Created: msg.Created}}
		return nil

	case []any:
		var payloads []map[string]any
		var records []*protomfx.Record
		for i, val := range p {
			v, ok := val.(map[string]any)
			if !ok {
				return errors.Wrap(ErrTransform, errInvalidNestedJSON)
			}

			ts, err := transformTimeField(v, transformer)
			if err != nil {
				return errors.Wrap(ErrInvalidTimeField, err)
			}

			formattedPayload := filterPayloadFields(v, transformer.DataFilters)
			data, err := json.Marshal(formattedPayload)
			if err != nil {
				return err
			}

			// Timestamps assigned per record by the publisher apply to items without a time field value.
			created := msg.Created
			if len(msg.Records) == len(p) && msg.Records[i].Created != 0 {
				created = msg.Records[i].Created
			}
			if ts != 0 {
				created = ts
			}

			payloads = append(payloads, formattedPayload)
			records = append(records, &protomfx.Record{Payload: data, Created: created})
		}

		data, err := json.Marshal(payloads)
		if err != nil {
			return err
		}
		msg.Payload = data
		msg.Records = records
		return nil

	default:
		return errors.Wrap(ErrTransform, errInvalidFormat)
	}
}

func transformTimeField(payload any, transformer domain.Transformer) (int64, error) {
	if transformer.TimeField == "" {
		return 0, nil
	}

	val := payload
	keys := strings.Split(transformer.TimeField, ".")

	for _, k := range keys {
		switch v := val.(type) {
		case map[string]any:
			current, exists := v[k]
			if !exists {
				return 0, nil
			}
			val = current

		case []any:
			if len(v) == 0 {
				return 0, nil
			}
			firstItem, ok := v[0].(map[string]any)
			if !ok {
				return 0, nil
			}
			current, exists := firstItem[k]
			if !exists {
				return 0, nil
			}
			val = current

		default:
			return 0, nil
		}
	}

	return ParseTimestamp(transformer, val)
}

func extractPayload(payload any, dataField string) any {
	if dataField == "" {
		return payload
	}

	value := payload
	keys := strings.Split(dataField, ".")

	for _, k := range keys {
		switch v := value.(type) {
		case map[string]any:
			current, exists := v[k]
			if !exists {
				return nil
			}
			value = current

		case []any:
			value = extractItemsValue(v, k)

		default:
			return nil
		}
	}

	return value
}

func extractItemsValue(items []any, key string) any {
	var values []any
	for _, entry := range items {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil
		}

		current, exists := item[key]
		if !exists {
			continue
		}

		// A single item keeps the object output that existing transformers produce.
		if len(items) == 1 {
			return current
		}

		if arr, ok := current.([]any); ok {
			values = append(values, arr...)
			continue
		}
		values = append(values, current)
	}

	if len(values) == 0 {
		return nil
	}

	return values
}

func filterPayloadFields(payload map[string]any, dataFilters []string) map[string]any {
	if len(dataFilters) == 0 {
		return payload
	}

	filteredPayload := make(map[string]any)

	for _, key := range dataFilters {
		// Split nested path
		keys := strings.Split(key, ".")
		var value any = payload

		// Traverse nested structure
		for _, k := range keys {
			current, ok := value.(map[string]any)
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
