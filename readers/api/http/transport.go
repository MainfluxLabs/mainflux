// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api/http/backup"
	"github.com/MainfluxLabs/mainflux/readers/api/http/messages"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.Service, tracer opentracing.Tracer, svcName string, logger log.Logger) http.Handler {
	mux := bone.New()
	mux = messages.MakeHandler(svc, mux, tracer, logger)
	mux = backup.MakeHandler(svc, mux, tracer, logger)
	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
