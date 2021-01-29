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
