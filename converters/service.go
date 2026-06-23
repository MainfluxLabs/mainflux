// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package converters

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	protocol = "http"

	maxBatchBytes = 1 << 20 // 1MB
	batchDelay    = 1 * time.Second
)

// ErrInvalidTimeField represents an invalid timestamp.
var ErrInvalidTimeField = errors.New("invalid time field")

// reservedFields are message keys that must not appear inside the payload.
var reservedFields = map[string]bool{
	"protocol":  true,
	"publisher": true,
	"subtopic":  true,
	"created":   true,
}

// Service specifies coap service API.
type Service interface {
	PublishSenMLMessagesFromCSV(ctx context.Context, key string, csvLines [][]string) error
	PublishJSONMessagesFromCSV(ctx context.Context, key string, csvLines [][]string) error
	PublishSenMLMessagesFromJSON(ctx context.Context, key string, records []map[string]any) error
	PublishJSONMessagesFromJSON(ctx context.Context, key string, records []map[string]any) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    domain.ThingsClient
}

// New instantiates the HTTP adapter implementation.
func New(pub messaging.Publisher, things domain.ThingsClient) Service {
	return &adapterService{
		publisher: pub,
		things:    things,
	}
}

func (as *adapterService) PublishSenMLMessagesFromCSV(ctx context.Context, key string, csvLines [][]string) error {
	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	headers := csvLines[0]
	msgs := []map[string]any{}
	size := 0

	flush := func() error {
		if len(msgs) == 0 {
			return nil
		}
		data, err := json.Marshal(msgs)
		if err != nil {
			return err
		}
		msg.Payload = data
		if _, err = as.publish(ctx, key, msg); err != nil {
			return err
		}
		msgs = []map[string]any{}
		size = 0
		return nil
	}

	for i := 1; i < len(csvLines); i++ {
		row := csvLines[i]
		record := map[string]any{}
		for j, col := range headers {
			if j >= len(row) || row[j] == "" {
				continue
			}
			val := row[j]
			switch col {
			case "t", "time", "v", "value", "s", "sum":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					record[col] = f
				}
			case "vb", "bool_value":
				if b, err := strconv.ParseBool(val); err == nil {
					record[col] = b
				}
			case "n", "name", "vs", "string_value", "vd", "data_value", "u", "unit":
				record[col] = val
			case "protocol":
				if val != "" {
					msg.Protocol = val
				}
			case "subtopic":
				msg.Subtopic = val
			}
		}
		entries, err := toSenMLEntries(record)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entryData, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			msgs = append(msgs, entry)
			size += len(entryData)
			if size >= maxBatchBytes {
				if err := flush(); err != nil {
					return err
				}
				time.Sleep(batchDelay)
			}
		}
	}

	return flush()
}

func (as *adapterService) PublishJSONMessagesFromCSV(ctx context.Context, key string, csvLines [][]string) error {
	thKey := domain.ThingKey{
		Value: key,
		Type:  domain.KeyTypeInternal,
	}

	pc, err := as.things.GetPubConfigByKey(ctx, thKey)
	if err != nil {
		return err
	}

	timeField := pc.ProfileConfig.Transformer.TimeField

	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	keys := csvLines[0][1:]
	msgs := []map[string]any{}
	size := 0
	timeFieldIdx := slices.Index(csvLines[0], timeField)
	for i := 1; i < len(csvLines); i++ {
		record := map[string]any{}
		if timeField != "" && timeFieldIdx != -1 {
			t, err := strconv.ParseFloat(csvLines[i][timeFieldIdx], 64)
			if err != nil {
				return ErrInvalidTimeField
			}
			record["Created"] = t
		}
		for j, columnName := range keys {
			if reservedFields[columnName] {
				switch columnName {
				case "protocol":
					if val := csvLines[i][j+1]; val != "" {
						msg.Protocol = val
					}
				case "subtopic":
					msg.Subtopic = csvLines[i][j+1]
				}
				continue
			}
			if f, err := strconv.ParseFloat(csvLines[i][j+1], 64); err == nil {
				record[columnName] = f
			} else {
				record[columnName] = csvLines[i][j+1]
			}
		}
		recData, err := json.Marshal(record)
		if err != nil {
			return err
		}
		msgs = append(msgs, record)
		size += len(recData)
		if size >= maxBatchBytes {
			data, err := json.Marshal(msgs)
			if err != nil {
				return err
			}
			msg.Payload = data
			if _, err = as.publish(ctx, key, msg); err != nil {
				return err
			}
			msgs = []map[string]any{}
			size = 0
			time.Sleep(batchDelay)
		}
	}
	if len(msgs) > 0 {
		data, err := json.Marshal(msgs)
		if err != nil {
			return err
		}
		msg.Payload = data
		if _, err := as.publish(ctx, key, msg); err != nil {
			return err
		}
	}
	return nil
}

func (as *adapterService) PublishSenMLMessagesFromJSON(ctx context.Context, key string, records []map[string]any) error {
	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	msgs := []map[string]any{}
	size := 0

	flush := func() error {
		if len(msgs) == 0 {
			return nil
		}
		data, err := json.Marshal(msgs)
		if err != nil {
			return err
		}
		msg.Payload = data
		if _, err = as.publish(ctx, key, msg); err != nil {
			return err
		}
		msgs = []map[string]any{}
		size = 0
		return nil
	}

	for _, record := range records {
		if s, ok := record["protocol"].(string); ok && s != "" {
			msg.Protocol = s
		}
		if s, ok := record["subtopic"].(string); ok {
			msg.Subtopic = s
		}
		entries, err := toSenMLEntries(record)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entryData, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			msgs = append(msgs, entry)
			size += len(entryData)
			if size >= maxBatchBytes {
				if err := flush(); err != nil {
					return err
				}
				time.Sleep(batchDelay)
			}
		}
	}

	return flush()
}

// toSenMLEntries converts one input record into SenML measurement entries.
// Accepts both standard SenML keys (t, n, v, vs, vd, vb, u, s) and
// reader-export aliases (time, name, value, string_value, data_value, bool_value, unit, sum).
func toSenMLEntries(record map[string]any) ([]map[string]any, error) {
	var t float64
	if v, ok := record["t"].(float64); ok {
		t = v
	} else if v, ok := record["time"].(float64); ok {
		t = v / 1e9 // reader export is nanoseconds; SenML t field is seconds
	} else {
		return nil, ErrInvalidTimeField
	}

	n, _ := record["n"].(string)
	if n == "" {
		n, _ = record["name"].(string)
	}

	if n != "" {
		entry := map[string]any{"n": n, "t": t}
		for k, v := range record {
			switch k {
			case "v", "value":
				if f, ok := v.(float64); ok {
					entry["v"] = f
				}
			case "vs", "string_value":
				if s, ok := v.(string); ok {
					entry["vs"] = s
				}
			case "vd", "data_value":
				if s, ok := v.(string); ok {
					entry["vd"] = s
				}
			case "vb", "bool_value":
				if b, ok := v.(bool); ok {
					entry["vb"] = b
				}
			case "u", "unit":
				if s, ok := v.(string); ok {
					entry["u"] = s
				}
			case "s", "sum":
				if f, ok := v.(float64); ok {
					entry["s"] = f
				}
			}
		}
		return []map[string]any{entry}, nil
	}

	var entries []map[string]any
	for k, val := range record {
		if k == "t" || k == "time" {
			continue
		}
		v, ok := val.(float64)
		if !ok {
			continue // non-numeric fields have no SenML representation in multi-measurement mode
		}
		entries = append(entries, map[string]any{"n": k, "v": v, "t": t})
	}
	return entries, nil
}

func (as *adapterService) PublishJSONMessagesFromJSON(ctx context.Context, key string, records []map[string]any) error {
	thKey := domain.ThingKey{
		Value: key,
		Type:  domain.KeyTypeInternal,
	}

	pc, err := as.things.GetPubConfigByKey(ctx, thKey)
	if err != nil {
		return err
	}

	timeField := pc.ProfileConfig.Transformer.TimeField

	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	msgs := []map[string]any{}
	size := 0
	for _, inputRecord := range records {
		// Auto-unwrap reader-exported Format A: {"payload":{...},"created":...,...}
		source := inputRecord
		unwrapped := false
		var outerTimestamp float64
		if pld, ok := inputRecord["payload"].(map[string]any); ok {
			source = pld
			unwrapped = true
			// Prefer the outer timeField value (numeric); fall back to outer "created".
			if timeField != "" {
				outerTimestamp, _ = inputRecord[timeField].(float64)
			}
			if outerTimestamp == 0 {
				outerTimestamp, _ = inputRecord["created"].(float64)
			}
			// Preserve envelope fields from the outer record into the message.
			if s, ok := inputRecord["protocol"].(string); ok && s != "" {
				msg.Protocol = s
			}
			if s, ok := inputRecord["subtopic"].(string); ok {
				msg.Subtopic = s
			}
		}

		record := map[string]any{}
		if outerTimestamp != 0 {
			record["Created"] = outerTimestamp
		}

		for k, v := range source {
			// For flat records, envelope fields are lifted into the message rather
			// than stored in the payload. For unwrapped Format A the outer envelope
			// is already handled above; inner fields are sensor data.
			if !unwrapped && reservedFields[k] {
				switch k {
				case "protocol":
					if s, ok := v.(string); ok && s != "" {
						msg.Protocol = s
					}
				case "subtopic":
					if s, ok := v.(string); ok {
						msg.Subtopic = s
					}
				case "created":
					if t, ok := v.(float64); ok && t != 0 {
						record["Created"] = t
					}
				}
				continue
			}
			if timeField != "" && k == timeField {
				if t, ok := v.(float64); ok {
					record["Created"] = t
				}
			}
			record[k] = v
		}
		recData, err := json.Marshal(record)
		if err != nil {
			return err
		}
		msgs = append(msgs, record)
		size += len(recData)
		if size >= maxBatchBytes {
			data, err := json.Marshal(msgs)
			if err != nil {
				return err
			}
			msg.Payload = data
			if _, err = as.publish(ctx, key, msg); err != nil {
				return err
			}
			msgs = []map[string]any{}
			size = 0
			time.Sleep(batchDelay)
		}
	}
	if len(msgs) > 0 {
		data, err := json.Marshal(msgs)
		if err != nil {
			return err
		}
		msg.Payload = data
		if _, err := as.publish(ctx, key, msg); err != nil {
			return err
		}
	}
	return nil
}

func (as *adapterService) publish(ctx context.Context, key string, msg protomfx.Message) (m protomfx.Message, err error) {
	pcr := domain.ThingKey{Type: domain.KeyTypeInternal, Value: key}

	pc, err := as.things.GetPubConfigByKey(ctx, pcr)
	if err != nil {
		return protomfx.Message{}, err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return protomfx.Message{}, err
	}

	for _, subject := range nats.GetPublishSubjects(msg.Publisher, msg.Subtopic, pc.ProfileConfig) {
		if err := as.publisher.Publish(subject, msg); err != nil {
			return protomfx.Message{}, err
		}
	}

	return m, nil
}
