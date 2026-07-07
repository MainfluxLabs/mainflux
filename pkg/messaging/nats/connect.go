// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"errors"
	"strings"

	broker "github.com/nats-io/nats.go"
)

// streamName is the name of the JetStream stream backing all Mainflux subjects.
const streamName = "mainflux"

var streamSubjects = []string{
	SubjectThings,
	SubjectGroups,
	SubjectAlarms,
	SubjectSmtp,
	SubjectSmpp,
	SubjectWebhooks,
	SubjectRules,
}

func connect(url string) (*broker.Conn, broker.JetStreamContext, error) {
	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	if err := ensureStream(js); err != nil {
		conn.Close()
		return nil, nil, err
	}

	return conn, js, nil
}

func ensureStream(js broker.JetStreamContext) error {
	switch _, err := js.StreamInfo(streamName); {
	case err == nil:
		return nil
	case !errors.Is(err, broker.ErrStreamNotFound):
		return err
	}

	_, err := js.AddStream(&broker.StreamConfig{
		Name:     streamName,
		Subjects: streamSubjects,
		Storage:  broker.FileStorage,
	})

	if err != nil && strings.Contains(err.Error(), "already in use") {
		return nil
	}
	return err
}

func durableName(queue, id, topic string) string {
	base := id
	if queue != "" {
		base = queue
	}
	return sanitizeDurable(base + "_" + topic)
}

var durableReplacer = strings.NewReplacer(
	".", "_",
	"*", "all",
	">", "gt",
	" ", "_",
	"/", "_",
	"\\", "_",
)

func sanitizeDurable(s string) string {
	return durableReplacer.Replace(s)
}
