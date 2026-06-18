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
	"Created":   true,
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
	keys := csvLines[0][1:]
	msgs := []map[string]any{}
	size := 0
	for i := 1; i < len(csvLines); i++ {
		t, err := strconv.ParseFloat(csvLines[i][0], 64)
		if err != nil {
			return ErrInvalidTimeField
		}
		for j, n := range keys {
			v, err := strconv.ParseFloat(csvLines[i][j+1], 64)
			if err != nil {
				return err
			}
			rec := map[string]any{"n": n, "v": v, "t": t}
			recData, err := json.Marshal(rec)
			if err != nil {
				return err
			}
			msgs = append(msgs, rec)
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
	counter := 0
	msgs := []map[string]any{}
	for i, record := range records {
		tVal, ok := record["t"]
		if !ok {
			return ErrInvalidTimeField
		}
		t, ok := tVal.(float64)
		if !ok {
			return ErrInvalidTimeField
		}
		for n, val := range record {
			if n == "t" {
				continue
			}
			v, ok := val.(float64)
			if !ok {
				return ErrInvalidTimeField
			}
			msgs = append(msgs, map[string]any{"n": n, "v": v, "t": t})
			counter++
			if counter >= 50000 {
				data, err := json.Marshal(msgs)
				if err != nil {
					return err
				}
				msg.Payload = data
				if _, err = as.publish(ctx, key, msg); err != nil {
					return err
				}
				counter = 0
				msgs = []map[string]any{}
				time.Sleep(30 * time.Second)
			}
		}
		if i == len(records)-1 && len(msgs) > 0 {
			data, err := json.Marshal(msgs)
			if err != nil {
				return err
			}
			msg.Payload = data
			if _, err = as.publish(ctx, key, msg); err != nil {
				return err
			}
		}
	}
	return nil
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
	counter := 0
	msgs := []map[string]any{}
	for i, inputRecord := range records {
		// Auto-unwrap reader-exported Format A: {"payload":{...},"created":...,...}
		source := inputRecord
		var outerTimestamp float64
		if pld, ok := inputRecord["payload"].(map[string]any); ok {
			source = pld
			// Prefer the outer timeField value (numeric); fall back to outer "created".
			if timeField != "" {
				outerTimestamp, _ = inputRecord[timeField].(float64)
			}
			if outerTimestamp == 0 {
				outerTimestamp, _ = inputRecord["created"].(float64)
			}
		}

		record := map[string]any{}
		if outerTimestamp != 0 {
			record["Created"] = outerTimestamp
		}

		for k, v := range source {
			if reservedFields[k] {
				continue
			}
			if timeField != "" && k == timeField {
				if t, ok := v.(float64); ok {
					record["Created"] = t
				}
			}
			record[k] = v
		}
		msgs = append(msgs, record)
		counter++
		if counter == 50000 || i == len(records)-1 {
			data, err := json.Marshal(msgs)
			if err != nil {
				return err
			}
			msg.Payload = data
			_, err = as.publish(ctx, key, msg)
			if err != nil {
				return err
			}
			counter = 0
			msgs = []map[string]any{}
			if i != len(records)-1 {
				time.Sleep(30 * time.Second)
			}
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
