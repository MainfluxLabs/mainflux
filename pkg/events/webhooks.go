// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import "encoding/json"

// WebhookCreated signals the creation of a webhook, without its URL and headers, which may hold credentials.
type WebhookCreated struct {
	ID       string
	ThingID  string
	Name     string
	Metadata map[string]any
}

func (e WebhookCreated) Operation() string {
	return WebhookCreate
}

func (e WebhookCreated) Encode() map[string]any {
	m := encodeDescribed(e.ID, e.Name, "", e.Metadata)
	m["thing_id"] = e.ThingID
	return m
}

func decodeWebhookCreated(e redisEvent) WebhookCreated {
	wh := WebhookCreated{
		ID:      e.field("id", ""),
		ThingID: e.field("thing_id", ""),
		Name:    e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &wh.Metadata)
	}
	return wh
}

// WebhookUpdated signals a webhook update, without its URL and headers, which may hold credentials.
type WebhookUpdated struct {
	ID       string
	Name     string
	Metadata map[string]any
}

func (e WebhookUpdated) Operation() string {
	return WebhookUpdate
}

func (e WebhookUpdated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, "", e.Metadata)
}

func decodeWebhookUpdated(e redisEvent) WebhookUpdated {
	wh := WebhookUpdated{
		ID:   e.field("id", ""),
		Name: e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &wh.Metadata)
	}
	return wh
}

// WebhookRemoved signals that a webhook has been removed.
type WebhookRemoved struct {
	ID string
}

func (e WebhookRemoved) Operation() string {
	return WebhookRemove
}

func (e WebhookRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeWebhookRemoved(e redisEvent) WebhookRemoved {
	return WebhookRemoved{ID: e.field("id", "")}
}
