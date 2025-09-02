// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"context"
	"errors"
)

const (
	// EqualKey represents the equal comparison operator key.
	EqualKey = "eq"
	// LowerThanKey represents the lower-than comparison operator key.
	LowerThanKey = "lt"
	// LowerThanEqualKey represents the lower-than-or-equal comparison operator key.
	LowerThanEqualKey = "le"
	// GreaterThanKey represents the greater-than-or-equal comparison operator key.
	GreaterThanKey = "gt"
	// GreaterThanEqualKey represents the greater-than-or-equal comparison operator key.
	GreaterThanEqualKey = "ge"
	// AggregationMin represents the minimum aggregation key.
	AggregationMin = "min"
	// AggregationMax represents the maximum aggregation key.
	AggregationMax = "max"
	// AggregationAvg represents the average aggregation key.
	AggregationAvg = "avg"
	// AggregationCount represents the count aggregation key.
	AggregationCount = "count"
)

// ErrReadMessages indicates failure occurred while reading messages from database.
var ErrReadMessages = errors.New("failed to read messages from database")

// MessageRepository specifies message reader API.
type MessageRepository interface {
	ListJSONMessages(rpm JSONMetadata) (MessagesPage, error)
	ListSenMLMessages(rpm SenMLMetadata) (MessagesPage, error)

	BackupJSONMessages(rpm JSONMetadata) (MessagesPage, error)
	BackupSenMLMessages(rpm SenMLMetadata) (MessagesPage, error)

	RestoreJSONMessages(ctx context.Context, messages ...Message) error
	RestoreSenMLMessageS(ctx context.Context, messages ...Message) error

	DeleteJSONMessages(ctx context.Context, rpm JSONMetadata) error
	DeleteSenMLMessages(ctx context.Context, rpm SenMLMetadata) error
}

// Message represents any message format.
type Message interface{}

// MessagesPage contains page related metadata as well as list of messages that
// belong to this page.
type MessagesPage struct {
	JSONMetadata  `json:"json_metadata, omitempty"`
	SenMLMetadata `json:"senml_metadata, omitempty"`
	Total         uint64
	Messages      []Message
}

// PageMetadata represents the parameters used to create database queries
type PageMetadata struct {
	Offset      uint64  `json:"offset"`
	Limit       uint64  `json:"limit"`
	Subtopic    string  `json:"subtopic,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	Protocol    string  `json:"protocol,omitempty"`
	Name        string  `json:"name,omitempty"`
	Value       float64 `json:"v,omitempty"`
	Comparator  string  `json:"comparator,omitempty"`
	BoolValue   bool    `json:"vb,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	From        int64   `json:"from,omitempty"`
	To          int64   `json:"to,omitempty"`
	Format      string  `json:"format,omitempty"`
	AggInterval string  `json:"agg_interval,omitempty"`
	AggType     string  `json:"agg_type,omitempty"`
	AggField    string  `json:"agg_field,omitempty"`
}

// SenMLMetadata represents the parameters used to create database queries
type SenMLMetadata struct {
	Offset      uint64  `json:"offset"`
	Limit       uint64  `json:"limit"`
	Subtopic    string  `json:"subtopic,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	Protocol    string  `json:"protocol,omitempty"`
	Name        string  `json:"name,omitempty"`
	Value       float64 `json:"v,omitempty"`
	Comparator  string  `json:"comparator,omitempty"`
	BoolValue   bool    `json:"vb,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	From        int64   `json:"from,omitempty"`
	To          int64   `json:"to,omitempty"`
	Format      string  `json:"format,omitempty"`
	AggInterval string  `json:"agg_interval,omitempty"`
	AggType     string  `json:"agg_type,omitempty"`
	AggField    string  `json:"agg_field,omitempty"`
}

// JSONMetadata represents the parameters used to create database queries
type JSONMetadata struct {
	Offset      uint64 `json:"offset"`
	Limit       uint64 `json:"limit"`
	Subtopic    string `json:"subtopic,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	From        int64  `json:"from,omitempty"`
	To          int64  `json:"to,omitempty"`
	AggInterval string `json:"agg_interval,omitempty"`
	AggType     string `json:"agg_type,omitempty"`
	AggField    string `json:"agg_field,omitempty"`
}

// ParseValueComparator convert comparison operator keys into mathematic anotation
func ParseValueComparator(query map[string]interface{}) string {
	comparator := "="
	val, ok := query["comparator"]
	if ok {
		switch val.(string) {
		case EqualKey:
			comparator = "="
		case LowerThanKey:
			comparator = "<"
		case LowerThanEqualKey:
			comparator = "<="
		case GreaterThanKey:
			comparator = ">"
		case GreaterThanEqualKey:
			comparator = ">="
		}
	}

	return comparator
}
