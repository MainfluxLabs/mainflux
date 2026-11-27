// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import "encoding/json"

// DownlinkCreated signals the creation of a downlink, without its URL, headers and payload, which may hold credentials.
type DownlinkCreated struct {
	ID       string
	ThingID  string
	Name     string
	Metadata map[string]any
}

func (e DownlinkCreated) Operation() string {
	return DownlinkCreate
}

func (e DownlinkCreated) Encode() map[string]any {
	m := encodeDescribed(e.ID, e.Name, "", e.Metadata)
	m["thing_id"] = e.ThingID
	return m
}

func decodeDownlinkCreated(e redisEvent) DownlinkCreated {
	dl := DownlinkCreated{
		ID:      e.field("id", ""),
		ThingID: e.field("thing_id", ""),
		Name:    e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &dl.Metadata)
	}
	return dl
}

// DownlinkUpdated signals a downlink update, without its URL, headers and payload, which may hold credentials.
type DownlinkUpdated struct {
	ID       string
	Name     string
	Metadata map[string]any
}

func (e DownlinkUpdated) Operation() string {
	return DownlinkUpdate
}

func (e DownlinkUpdated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, "", e.Metadata)
}

func decodeDownlinkUpdated(e redisEvent) DownlinkUpdated {
	dl := DownlinkUpdated{
		ID:   e.field("id", ""),
		Name: e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &dl.Metadata)
	}
	return dl
}

// DownlinkRemoved signals that a downlink has been removed.
type DownlinkRemoved struct {
	ID string
}

func (e DownlinkRemoved) Operation() string {
	return DownlinkRemove
}

func (e DownlinkRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeDownlinkRemoved(e redisEvent) DownlinkRemoved {
	return DownlinkRemoved{ID: e.field("id", "")}
}
