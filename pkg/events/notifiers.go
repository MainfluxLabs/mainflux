// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"strings"
)

// NotifierCreated signals the creation of a notifier.
type NotifierCreated struct {
	ID       string
	Name     string
	Contacts []string
	Metadata map[string]any
}

func (e NotifierCreated) Operation() string {
	return NotifierCreate
}

func (e NotifierCreated) Encode() map[string]any {
	return encodeNotifier(e.ID, e.Name, e.Contacts, e.Metadata)
}

func decodeNotifierCreated(e redisEvent) NotifierCreated {
	id, name, contacts, metadata := decodeNotifier(e)
	return NotifierCreated{
		ID:       id,
		Name:     name,
		Contacts: contacts,
		Metadata: metadata,
	}
}

// NotifierUpdated signals a notifier update.
type NotifierUpdated struct {
	ID       string
	Name     string
	Contacts []string
	Metadata map[string]any
}

func (e NotifierUpdated) Operation() string {
	return NotifierUpdate
}

func (e NotifierUpdated) Encode() map[string]any {
	return encodeNotifier(e.ID, e.Name, e.Contacts, e.Metadata)
}

func decodeNotifierUpdated(e redisEvent) NotifierUpdated {
	id, name, contacts, metadata := decodeNotifier(e)
	return NotifierUpdated{
		ID:       id,
		Name:     name,
		Contacts: contacts,
		Metadata: metadata,
	}
}

// NotifierRemoved signals that a notifier has been removed.
type NotifierRemoved struct {
	ID string
}

func (e NotifierRemoved) Operation() string {
	return NotifierRemove
}

func (e NotifierRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeNotifierRemoved(e redisEvent) NotifierRemoved {
	return NotifierRemoved{ID: e.field("id", "")}
}

func encodeNotifier(id, name string, contacts []string, metadata map[string]any) map[string]any {
	m := encodeDescribed(id, name, "", metadata)
	if len(contacts) > 0 {
		m["contacts"] = strings.Join(contacts, ",")
	}
	return m
}

func decodeNotifier(e redisEvent) (id, name string, contacts []string, metadata map[string]any) {
	id = e.field("id", "")
	name = e.field("name", "")
	if raw := e.field("contacts", ""); raw != "" {
		contacts = strings.Split(raw, ",")
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &metadata)
	}
	return id, name, contacts, metadata
}
