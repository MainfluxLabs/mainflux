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
		if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
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
			if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
				return err
			}
			msgs = []map[string]any{}
			size = 0
			time.Sleep(batchDelay)
		}
	}
	if len(msgs) > 0 {
		if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
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
		if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
			return err
		}
		msgs = []map[string]any{}
		size = 0
		return nil
	}

	for _, record := range records {
		applyEnvelopeFields(&msg, record)
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
		record, err := parseJSONRecord(inputRecord, timeField, &msg)
		if err != nil {
			return err
		}
		recData, err := json.Marshal(record)
		if err != nil {
			return err
		}
		msgs = append(msgs, record)
		size += len(recData)
		if size >= maxBatchBytes {
			if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
				return err
			}
			msgs = []map[string]any{}
			size = 0
			time.Sleep(batchDelay)
		}
	}
	if len(msgs) > 0 {
		if err := as.publishMsgs(ctx, key, msgs, msg); err != nil {
			return err
		}
	}
	return nil
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

	n := ""
	if s, ok := record["n"].(string); ok {
		n = s
	}
	if n == "" {
		if s, ok := record["name"].(string); ok {
			n = s
		}
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
				if s, ok := v.(string); ok && s != "" {
					entry["vs"] = s
				}
			case "vd", "data_value":
				if s, ok := v.(string); ok && s != "" {
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

// applyEnvelopeFields sets msg.Protocol and msg.Subtopic from a source map.
func applyEnvelopeFields(msg *protomfx.Message, src map[string]any) {
	if s, ok := src["protocol"].(string); ok && s != "" {
		msg.Protocol = s
	}
	if s, ok := src["subtopic"].(string); ok {
		msg.Subtopic = s
	}
}

// parseJSONRecord converts one input record into a payload record and updates
// msg.Protocol / msg.Subtopic from envelope fields. Handles both Format A
// (reader-exported {"payload":{...},"created":...}) and flat records.
func parseJSONRecord(inputRecord map[string]any, timeField string, msg *protomfx.Message) (map[string]any, error) {
	record := map[string]any{}

	if pld, ok := inputRecord["payload"].(map[string]any); ok {
		// Format A — unwrap outer envelope, prefer timeField over "created".
		var ts float64
		if timeField != "" {
			if v, exists := inputRecord[timeField]; exists {
				t, ok := v.(float64)
				if !ok {
					return nil, ErrInvalidTimeField
				}
				ts = t
			}
		}
		if ts == 0 {
			if v, exists := inputRecord["created"]; exists {
				t, ok := v.(float64)
				if !ok {
					return nil, ErrInvalidTimeField
				}
				ts = t
			}
		}
		if ts != 0 {
			record["Created"] = ts
		}
		applyEnvelopeFields(msg, inputRecord)
		for k, v := range pld {
			if timeField != "" && k == timeField {
				t, ok := v.(float64)
				if !ok {
					return nil, ErrInvalidTimeField
				}
				record["Created"] = t
			}
			record[k] = v
		}
		return record, nil
	}

	// Flat record — lift envelope fields into msg, store the rest in record.
	for k, v := range inputRecord {
		if reservedFields[k] {
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
				t, ok := v.(float64)
				if !ok {
					return nil, ErrInvalidTimeField
				}
				if t != 0 {
					record["Created"] = t
				}
			}
			continue
		}
		if timeField != "" && k == timeField {
			t, ok := v.(float64)
			if !ok {
				return nil, ErrInvalidTimeField
			}
			record["Created"] = t
		}
		record[k] = v
	}
	return record, nil
}

// publishMsgs marshals a batch of records, sets msg.Payload, and publishes.
func (as *adapterService) publishMsgs(ctx context.Context, key string, msgs []map[string]any, msg protomfx.Message) error {
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}
	msg.Payload = data
	return as.publish(ctx, key, msg)
}

func (as *adapterService) publish(ctx context.Context, key string, msg protomfx.Message) error {
	pcr := domain.ThingKey{Type: domain.KeyTypeInternal, Value: key}

	pc, err := as.things.GetPubConfigByKey(ctx, pcr)
	if err != nil {
		return err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	for _, subject := range nats.GetPublishSubjects(msg.Publisher, msg.Subtopic, pc.ProfileConfig) {
		if err := as.publisher.Publish(subject, msg); err != nil {
			return err
		}
	}

	return nil
}
