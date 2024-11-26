// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package json

// Payload represents JSON Message payload.
type Payload map[string]interface{}

// Config represents JSON Message Profile Config.
type Config map[string]interface{}

// Message represents a JSON messages.
type Message struct {
	Created       int64   `json:"created,omitempty" db:"created" bson:"created"`
	Subtopic      string  `json:"subtopic,omitempty" db:"subtopic" bson:"subtopic,omitempty"`
	Publisher     string  `json:"publisher,omitempty" db:"publisher" bson:"publisher"`
	Protocol      string  `json:"protocol,omitempty" db:"protocol" bson:"protocol"`
	Payload       Payload `json:"payload,omitempty" db:"payload" bson:"payload,omitempty"`
	ProfileConfig Config  `json:"config,omitempty" db:"config" bson:"config,omitempty"`
}

// Messages represents a list of JSON messages.
type Messages struct {
	Data []Message
}
