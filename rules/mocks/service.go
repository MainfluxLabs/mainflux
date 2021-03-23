// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"log"
	"os"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/rules"
)

const (
	sql = "select * from stream where v > 1.2;"
)

func NewService(users map[string]string, channels map[string]string, kuiperURL string) rules.Service {
	// map[token]email
	auth := NewAuthServiceClient(users)
	// map[chanID]email
	things := NewThingsClient(channels)
	logger, err := logger.New(os.Stdout, "info")
	if err != nil {
		log.Fatalf(err.Error())
	}
	kuiper := NewKuiperSDK(kuiperURL)
	return rules.New(kuiper, auth, things, logger)
}

func CreateRule(id, channel string) rules.Rule {
	var rule rules.Rule

	rule.ID = id
	rule.SQL = sql
	rule.Actions = append(rule.Actions, struct {
		Mainflux rules.Action `json:"mainflux"`
	}{
		Mainflux: rules.Action{
			Channel: channel,
		},
	})

	return rule
}
