// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
)

// Command represents a JSON command.
type Command struct {
	Created     int64           `json:"created,omitempty" db:"created" bson:"created"`
	Subtopic    string          `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher   string          `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	RecipientID string          `json:"recipientID,omitempty" db:"recipient_id" bson:"recipientID,omitempty"`
	Protocol    string          `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Payload     json.RawMessage `json:"payload,omitempty" db:"payload" bson:"payload,omitempty"`
}
