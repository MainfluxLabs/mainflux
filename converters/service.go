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

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	protocol = "http"
)

// ErrInvalidTimeField represents an invalid timestamp.
var ErrInvalidTimeField = errors.New("invalid time field")

// Service specifies coap service API.
type Service interface {
	PublishSenMLMessages(ctx context.Context, key string, csvLines [][]string) error
	PublishJSONMessages(ctx context.Context, key string, csvLines [][]string) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    protomfx.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(pub messaging.Publisher, things protomfx.ThingsServiceClient) Service {
	return &adapterService{
		publisher: pub,
		things:    things,
	}
}

func (as *adapterService) PublishSenMLMessages(ctx context.Context, key string, csvLines [][]string) error {
	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	counter := 0
	keys := csvLines[0][1:]
	msgs := []map[string]any{}
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
			record := map[string]any{
				"n": n,
				"v": v,
				"t": t,
			}
			msgs = append(msgs, record)
			counter++
			if counter >= 50000 || i == len(csvLines)-1 {
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
				if i != len(csvLines)-1 {
					time.Sleep(30 * time.Second)
				}
			}
		}
	}
	return nil
}

func (as *adapterService) PublishJSONMessages(ctx context.Context, key string, csvLines [][]string) error {
	msg := protomfx.Message{
		Protocol: protocol,
		Created:  time.Now().UnixNano(),
	}
	counter := 0
	keys := csvLines[0][1:]
	msgs := []map[string]any{}
	createdIdx := slices.Index(csvLines[0], "created")
	if createdIdx == -1 {
		return ErrInvalidTimeField
	}

	for i := 1; i < len(csvLines); i++ {
		record := map[string]any{
			"created": csvLines[i][createdIdx],
		}
		for j, columnName := range keys {
			if f, err := strconv.ParseFloat(csvLines[i][j+1], 64); err == nil {
				record[columnName] = f
			} else {
				record[columnName] = csvLines[i][j+1]
			}
		}
		msgs = append(msgs, record)
		counter++
		if counter == 50000 || i == len(csvLines)-1 {
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
			if i != len(csvLines)-1 {
				time.Sleep(30 * time.Second)
			}
		}
	}
	return nil
}

func (as *adapterService) publish(ctx context.Context, key string, msg protomfx.Message) (m protomfx.Message, err error) {
	pcr := &protomfx.ThingKey{
		Type:  things.KeyTypeInternal,
		Value: key,
	}

	pc, err := as.things.GetPubConfigByKey(ctx, pcr)
	if err != nil {
		return protomfx.Message{}, err
	}

	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return protomfx.Message{}, err
	}

	msg.Subject = nats.GetMessagesSubject(msg.Publisher, msg.Subtopic)
	if err := as.publisher.Publish(msg); err != nil {
		return protomfx.Message{}, err
	}

	return m, nil
}
