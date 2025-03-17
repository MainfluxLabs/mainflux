// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	svcthings "github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/api/http/groups"
	"github.com/MainfluxLabs/mainflux/things/api/http/members"
	"github.com/MainfluxLabs/mainflux/things/api/http/profiles"
	"github.com/MainfluxLabs/mainflux/things/api/http/things"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc svcthings.Service, tracer opentracing.Tracer, logger log.Logger) http.Handler {
	mux := bone.New()
	mux = things.MakeHandler(svc, mux, tracer, logger)
	mux = profiles.MakeHandler(svc, mux, tracer, logger)
	mux = groups.MakeHandler(svc, mux, tracer, logger)
	mux = members.MakeHandler(svc, mux, tracer, logger)
	mux.GetFunc("/health", mainflux.Health("things"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux

}
