// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml

import (
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers"
	"github.com/MainfluxLabs/senml"
)

const (
	// JSON represents SenML in JSON format content type.
	JSON = "application/senml+json"
	// CBOR represents SenML in CBOR format content type.
	CBOR = "application/senml+cbor"
)

var (
	errDecode    = errors.New("failed to decode senml")
	errNormalize = errors.New("failed to normalize senml")
)

var formats = map[string]senml.Format{
	JSON: senml.JSON,
	CBOR: senml.CBOR,
}

// TransformPayload converts a SenML pack into one record per normalized SenML record.
// Records without a time value are assigned the created timestamp.
func TransformPayload(contentType string, payload []byte, created int64) ([]*protomfx.Record, error) {
	format, ok := formats[contentType]
	if !ok {
		format = formats[JSON]
	}

	raw, err := senml.Decode(payload, format)
	if err != nil {
		return nil, errors.Wrap(errDecode, err)
	}

	normalized, err := senml.Normalize(raw)
	if err != nil {
		return nil, errors.Wrap(errNormalize, err)
	}

	records := make([]*protomfx.Record, len(normalized.Records))
	for i, v := range normalized.Records {
		// Use reception timestamp if SenML message Time is missing
		var t int64
		switch v.Time {
		case 0:
			t = created
		default:
			ts, err := transformers.SecondsToNanos(v.Time)
			if err != nil {
				return nil, err
			}
			t = ts
		}

		data, err := json.Marshal(Message{
			Name:        v.Name,
			Unit:        v.Unit,
			Time:        t,
			UpdateTime:  v.UpdateTime,
			Value:       v.Value,
			BoolValue:   v.BoolValue,
			DataValue:   v.DataValue,
			StringValue: v.StringValue,
			Sum:         v.Sum,
		})
		if err != nil {
			return nil, err
		}
		records[i] = &protomfx.Record{
			Payload: data,
			Time:    t,
		}
	}

	return records, nil
}
