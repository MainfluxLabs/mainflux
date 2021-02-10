//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package rules

// Action stores mainflux sink plugin settings
type Action struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
	Subtopic string `json:"subtopic"`
}

// Rule represents data used to create kuiper rule
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

// RuleInfo is used to fetch rule status from kuiper
type RuleInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
