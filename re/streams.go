//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package re

const (
	FORMAT = "json"
	TYPE   = "mainflux"
)

type Stream struct {
	Name     string `json:"name,omitempty"`
	Row      string `json:"row"`
	Topic    string `json:"topic"`
	Subtopic string `json:"subtopic"`
	Host     string `json:"host"`
}
type StreamInfo struct {
	Name         string `json:"Name"`
	StreamFields []struct {
		Name      string `json:"Name"`
		FieldType string `json:"FieldType"`
	} `json:"StreamFields"`
	Options struct {
		DATASOURCE string `json:"DATASOURCE"`
		FORMAT     string `json:"FORMAT"`
	} `json:"Options"`
}
