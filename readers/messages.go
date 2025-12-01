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

type JSONMessageRepository interface {
	// Retrieve retrieves the json messages with given filters.
	Retrieve(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// Backup backups the json messages with given filters.
	Backup(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// Restore restores the json messages.
	Restore(ctx context.Context, messages ...Message) error

	// Remove deletes the json messages within a time range.
	Remove(ctx context.Context, rpm JSONPageMetadata) error
}

type SenMLMessageRepository interface {
	// Retrieve retrieves the senml messages with given filters.
	Retrieve(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// Backup backups the senml messages with given filters.
	Backup(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// Restore restores the senml messages.
	Restore(ctx context.Context, messages ...Message) error

	// Remove deletes the senml messages within a time range.
	Remove(ctx context.Context, rpm SenMLPageMetadata) error
}

// Message represents any message format.
type Message interface{}

type MessagesPage struct {
	Total    uint64
	Messages []Message
}

type JSONMessagesPage struct {
	JSONPageMetadata
	MessagesPage
}

type SenMLMessagesPage struct {
	SenMLPageMetadata
	MessagesPage
}

// SenMLPageMetadata represents the parameters used to create database queries
type SenMLPageMetadata struct {
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
	AggInterval string  `json:"agg_interval,omitempty"`
	AggValue    uint64  `json:"agg_value,omitempty"`
	AggType     string  `json:"agg_type,omitempty"`
	AggField    string  `json:"agg_field,omitempty"`
	Dir         string  `json:"dir,omitempty"`
}

// JSONPageMetadata represents the parameters used to create database queries
type JSONPageMetadata struct {
	Offset      uint64 `json:"offset"`
	Limit       uint64 `json:"limit"`
	Subtopic    string `json:"subtopic,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	From        int64  `json:"from,omitempty"`
	To          int64  `json:"to,omitempty"`
	Filter      string `json:"filter,omitempty"`
	AggInterval string `json:"agg_interval,omitempty"`
	AggValue    uint64 `json:"agg_value,omitempty"`
	AggType     string `json:"agg_type,omitempty"`
	AggField    string `json:"agg_field,omitempty"`
	Dir         string `json:"dir,omitempty"`
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
