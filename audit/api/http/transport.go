// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/audit"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MakeHandler(svc audit.Service, ac domain.AuthClient, tracer opentracing.Tracer, logger log.Logger) http.Handler {
	_ = svc
	_ = ac
	_ = tracer
	_ = logger

	mux := bone.New()
	mux.GetFunc("/health", mainflux.Health("audit"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
