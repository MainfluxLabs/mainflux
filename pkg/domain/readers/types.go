// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

// Message represents any message format.
type Message any

// MessagesPage contains a page of messages.
type MessagesPage struct {
	Total    uint64
	Messages []Message
}

// JSONMessagesPage contains a page of JSON messages.
type JSONMessagesPage struct {
	JSONPageMetadata
	MessagesPage
}

// SenMLMessagesPage contains a page of SenML messages.
type SenMLMessagesPage struct {
	SenMLPageMetadata
	MessagesPage
}

// SenMLPageMetadata represents the parameters used to create database queries.
type SenMLPageMetadata struct {
	Offset      uint64   `json:"offset"`
	Limit       uint64   `json:"limit"`
	Subtopic    string   `json:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Name        string   `json:"name,omitempty"`
	Value       float64  `json:"v,omitempty"`
	Comparator  string   `json:"comparator,omitempty"`
	BoolValue   bool     `json:"vb,omitempty"`
	StringValue string   `json:"vs,omitempty"`
	DataValue   string   `json:"vd,omitempty"`
	From        int64    `json:"from,omitempty"`
	To          int64    `json:"to,omitempty"`
	AggInterval string   `json:"agg_interval,omitempty"`
	AggValue    uint64   `json:"agg_value,omitempty"`
	AggType     string   `json:"agg_type,omitempty"`
	AggFields   []string `json:"agg_fields,omitempty"`
	Dir         string   `json:"dir,omitempty"`
}

// JSONPageMetadata represents the parameters used to create database queries.
type JSONPageMetadata struct {
	Offset      uint64   `json:"offset"`
	Limit       uint64   `json:"limit"`
	Subtopic    string   `json:"subtopic,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	From        int64    `json:"from,omitempty"`
	To          int64    `json:"to,omitempty"`
	Filter      string   `json:"filter,omitempty"`
	AggInterval string   `json:"agg_interval,omitempty"`
	AggValue    uint64   `json:"agg_value,omitempty"`
	AggType     string   `json:"agg_type,omitempty"`
	AggFields   []string `json:"agg_fields,omitempty"`
	Dir         string   `json:"dir,omitempty"`
}
