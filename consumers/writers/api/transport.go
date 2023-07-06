// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP API handler with health check and metrics.
func MakeHandler(svcName string) http.Handler {
	r := bone.New()
	r.GetFunc("/health", mainflux.Health(svcName))
	r.Handle("/metrics", promhttp.Handler())

	return r
}
