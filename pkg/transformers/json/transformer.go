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

// TransformPayload converts a JSON object into one record and a JSON array
// into one record per item, and reports whether the payload is a JSON array.
// Records without a time field value are assigned the created timestamp.
func TransformPayload(transformer domain.Transformer, payload []byte, created int64) ([]*protomfx.Record, bool, error) {
	var pld any
	if err := json.Unmarshal(payload, &pld); err != nil {
		return nil, false, errors.Wrap(ErrTransform, err)
	}

	switch p := extractPayload(pld, transformer.DataField).(type) {
	case map[string]any:
		record, err := transformRecord(p, transformer, created)
		if err != nil {
			return nil, false, err
		}
		return []*protomfx.Record{record}, false, nil

	case []any:
		var records []*protomfx.Record
		for _, val := range p {
			v, ok := val.(map[string]any)
			if !ok {
				return nil, false, errors.Wrap(ErrTransform, errInvalidNestedJSON)
			}

			record, err := transformRecord(v, transformer, created)
			if err != nil {
				return nil, false, err
			}
			records = append(records, record)
		}
		return records, true, nil

	default:
		return nil, false, errors.Wrap(ErrTransform, errInvalidFormat)
	}
}

func transformRecord(payload map[string]any, transformer domain.Transformer, created int64) (*protomfx.Record, error) {
	ts, err := transformTimeField(payload, transformer)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidTimeField, err)
	}
	if ts != 0 {
		created = ts
	}

	data, err := json.Marshal(filterPayloadFields(payload, transformer.DataFilters))
	if err != nil {
		return nil, err
	}

	return &protomfx.Record{
		Payload: data,
		Time:    created,
	}, nil
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
