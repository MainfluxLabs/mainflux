//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package re

type Action struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
	Subtopic string `json:"subtopic"`
}

type Rule struct {
	ID      string `json:"id"`
	SQL     string `json:"sql"`
	Actions []struct {
		Mainflux Action `json:"mainflux"`
	} `json:"actions"`
	Options struct {
		SendMetaToSink bool `json:"sendMetaToSink"`
	} `json:"options"`
}
