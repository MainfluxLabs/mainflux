// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import "encoding/json"

// Message represents a JSON messages.
type Message struct {
	Created   int64  `json:"created,omitempty" db:"created" bson:"created"`
	Subtopic  string `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher string `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	Protocol  string `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Payload   []byte `json:"payload,omitempty" db:"payload" bson:"payload,omitempty"`
}

func (msg Message) ToMap() (map[string]any, error) {
	ret := map[string]any{
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]any{},
	}
	pld := make(map[string]any)
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}
