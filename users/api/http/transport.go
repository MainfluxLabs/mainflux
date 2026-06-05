// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"regexp"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	svcusers "github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/api/http/backup"
	"github.com/MainfluxLabs/mainflux/users/api/http/invites"
	"github.com/MainfluxLabs/mainflux/users/api/http/users"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc svcusers.Service, ac domain.AuthClient, tracer opentracing.Tracer, logger log.Logger, passwordRegex *regexp.Regexp) http.Handler {
	mux := bone.New()
	mux = users.MakeHandler(svc, ac, mux, tracer, logger, passwordRegex)
	mux = invites.MakeHandler(svc, ac, mux, tracer, logger, passwordRegex)
	mux = backup.MakeHandler(svc, ac, mux, tracer, logger)
	mux.GetFunc("/health", mainflux.Health("users"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
