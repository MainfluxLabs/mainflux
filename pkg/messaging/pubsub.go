// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import "errors"

const (
	SenMLContentType = "application/senml+json"
	JSONContentType  = "application/json"
)

var (
	// ErrConnect indicates that connection to MQTT broker failed
	ErrConnect = errors.New("failed to connect to MQTT broker")

	// ErrInvalidContentType indicates an invalid Content-Type
	ErrInvalidContentType = errors.New("invalid content type")
)

// PubSub  represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}
