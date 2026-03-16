// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/rules"
	httprules "github.com/MainfluxLabs/mainflux/rules/api/http/rules"
	httpscripts "github.com/MainfluxLabs/mainflux/rules/api/http/scripts"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Rule API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc rules.Service, logger log.Logger) http.Handler {
	mux := bone.New()
	mux = httprules.MakeHandler(svc, mux, tracer, logger)
	mux = httpscripts.MakeHandler(svc, mux, tracer, logger)

	mux.GetFunc("/health", mainflux.Health("rules"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
