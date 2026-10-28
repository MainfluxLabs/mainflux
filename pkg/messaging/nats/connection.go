// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"errors"
	"slices"
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
	SubjectOTA,
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
	info, err := js.StreamInfo(streamName)
	if err == nil {
		return addMissingSubjects(js, info.Config)
	}

	if !errors.Is(err, broker.ErrStreamNotFound) {
		return err
	}

	_, err = js.AddStream(&broker.StreamConfig{
		Name:     streamName,
		Subjects: streamSubjects,
		Storage:  broker.FileStorage,
	})

	if err != nil && errors.Is(err, broker.ErrStreamNameAlreadyInUse) {
		return nil
	}

	return err
}

// addMissingSubjects extends a stream created by an older version, since JetStream rejects publishes to subjects outside the stream.
func addMissingSubjects(js broker.JetStreamContext, config broker.StreamConfig) error {
	missing := false
	for _, subject := range streamSubjects {
		if !slices.Contains(config.Subjects, subject) {
			config.Subjects = append(config.Subjects, subject)
			missing = true
		}
	}

	if !missing {
		return nil
	}

	_, err := js.UpdateStream(&config)
	return err
}

func durableName(queue, id, topic string) string {
	base := id
	if queue != "" {
		base = queue
	}

	return durableReplacer.Replace(base + "_" + topic)
}

var durableReplacer = strings.NewReplacer(
	".", "_",
	"*", "all",
	">", "gt",
	" ", "_",
	"/", "_",
	"\\", "_",
)
