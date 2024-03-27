//go:build !rabbitmq
// +build !rabbitmq

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package brokers

import (
	"log"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
)

const (
	// SubjectSenML represents subject to subscribe for the SenML messages.
	SubjectSenML = "channels.*.senml.>"
	// SubjectJSON represents subject to subscribe for the JSON messages.
	SubjectJSON = "channels.*.json.>"
	// SubjectSmtp represents subject to subscribe for the SMTP notifications.
	SubjectSmtp = "smtp"
	// SubjectSmpp represents subject to subscribe for the SMPP notifications.
	SubjectSmpp = "smpp"
	// SubjectWebhook represents subject to subscribe for sending the Webhooks.
	SubjectWebhook = "webhook"
)

func init() {
	log.Println("The binary was build using Nats as the message broker")
}

func NewPublisher(url string) (messaging.Publisher, error) {
	pb, err := nats.NewPublisher(url)
	if err != nil {
		return nil, err
	}
	return pb, nil

}

func NewPubSub(url, queue string, logger logger.Logger) (messaging.PubSub, error) {
	pb, err := nats.NewPubSub(url, queue, logger)
	if err != nil {
		return nil, err
	}
	return pb, nil
}
