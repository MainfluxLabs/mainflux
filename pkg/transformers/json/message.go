// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Message represents a JSON messages.
type Message struct {
	Created  int64           `json:"created,omitempty" db:"created" bson:"created"`
	Subtopic string          `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	ThingID  string          `json:"thing_id,omitempty" db:"thing_id" bson:"thing_id"`
	Protocol string          `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Payload  json.RawMessage `json:"payload,omitempty" db:"payload" bson:"payload,omitempty"`
}

func (msg Message) ToMap() (map[string]any, error) {
	ret := map[string]any{
		"created":  msg.Created,
		"subtopic": msg.Subtopic,
		"thing_id": msg.ThingID,
		"protocol": msg.Protocol,
		"payload":  map[string]any{},
	}
	pld := make(map[string]any)
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}

func (msg Message) ToProtoMessage() protomfx.Message {
	return protomfx.Message{
		Created:  msg.Created,
		Subtopic: msg.Subtopic,
		ThingId:  msg.ThingID,
		Protocol: msg.Protocol,
		Payload:  msg.Payload,
	}
}
