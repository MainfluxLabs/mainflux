// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/MainfluxLabs/mainflux/uiconfigs/api/http/backup"
	"github.com/MainfluxLabs/mainflux/uiconfigs/api/http/orgs"
	"github.com/MainfluxLabs/mainflux/uiconfigs/api/http/things"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc uiconfigs.Service, logger log.Logger) http.Handler {
	mux := bone.New()
	mux = orgs.MakeHandler(tracer, svc, mux, logger)
	mux = things.MakeHandler(tracer, svc, mux, logger)
	mux = backup.MakeHandler(tracer, svc, mux, logger)
	mux.GetFunc("/health", mainflux.Health("uiconfigs"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux

}
