// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"sync"

	"github.com/MainfluxLabs/mainflux/coap"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type connObserversContextKey struct{}

// observation identifies an observation on a connection.
type observation struct {
	token    string
	subtopic string
}

// connObservers holds the observations registered on one connection, which end when it closes.
type connObservers struct {
	mu        sync.Mutex
	observers map[observation]coap.Client
}

// registryMu makes looking up and creating a connection's observers one step.
var registryMu sync.Mutex

// observersOf returns the observers of the connection, registering its close callback once,
// since the library keeps every close callback until the connection closes.
func observersOf(conn mux.Conn) *connObservers {
	registryMu.Lock()
	defer registryMu.Unlock()

	if observers, ok := conn.Context().Value(connObserversContextKey{}).(*connObservers); ok {
		return observers
	}

	observers := &connObservers{observers: make(map[observation]coap.Client)}
	conn.SetContextValue(connObserversContextKey{}, observers)
	conn.AddOnClose(observers.removeAll)

	return observers
}

func (co *connObservers) add(subtopic string, observer coap.Client) {
	co.mu.Lock()
	defer co.mu.Unlock()

	co.observers[observation{token: observer.Token(), subtopic: subtopic}] = observer
}

func (co *connObservers) remove(subtopic, token string) {
	co.mu.Lock()
	defer co.mu.Unlock()

	delete(co.observers, observation{token: token, subtopic: subtopic})
}

// removeAll ends the connection's observations, since a closed connection can't receive notifications.
func (co *connObservers) removeAll() {
	co.mu.Lock()
	observers := co.observers
	co.observers = make(map[observation]coap.Client)
	co.mu.Unlock()

	for obs, observer := range observers {
		_ = service.RemoveObserver(obs.subtopic, observer)
	}
}
