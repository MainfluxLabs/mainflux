// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/api/http/keys"
	"github.com/MainfluxLabs/mainflux/auth/api/http/orgs"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	mux := bone.New()
	mux = orgs.MakeHandler(svc, mux, tracer, logger)
	mux = keys.MakeHandler(svc, mux, tracer, logger)
	mux.GetFunc("/health", mainflux.Health("auth"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
