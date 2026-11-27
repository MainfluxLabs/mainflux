// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import "encoding/json"

// ModbusClientCreated signals the creation of a Modbus client.
type ModbusClientCreated struct {
	ID       string
	ThingID  string
	Name     string
	Metadata map[string]any
}

func (e ModbusClientCreated) Operation() string {
	return ModbusClientCreate
}

func (e ModbusClientCreated) Encode() map[string]any {
	m := encodeDescribed(e.ID, e.Name, "", e.Metadata)
	m["thing_id"] = e.ThingID
	return m
}

func decodeModbusClientCreated(e redisEvent) ModbusClientCreated {
	cl := ModbusClientCreated{
		ID:      e.field("id", ""),
		ThingID: e.field("thing_id", ""),
		Name:    e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &cl.Metadata)
	}
	return cl
}

// ModbusClientUpdated signals a Modbus client update.
type ModbusClientUpdated struct {
	ID       string
	Name     string
	Metadata map[string]any
}

func (e ModbusClientUpdated) Operation() string {
	return ModbusClientUpdate
}

func (e ModbusClientUpdated) Encode() map[string]any {
	return encodeDescribed(e.ID, e.Name, "", e.Metadata)
}

func decodeModbusClientUpdated(e redisEvent) ModbusClientUpdated {
	cl := ModbusClientUpdated{
		ID:   e.field("id", ""),
		Name: e.field("name", ""),
	}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &cl.Metadata)
	}
	return cl
}

// ModbusClientRemoved signals that a Modbus client has been removed.
type ModbusClientRemoved struct {
	ID string
}

func (e ModbusClientRemoved) Operation() string {
	return ModbusClientRemove
}

func (e ModbusClientRemoved) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeModbusClientRemoved(e redisEvent) ModbusClientRemoved {
	return ModbusClientRemoved{ID: e.field("id", "")}
}
