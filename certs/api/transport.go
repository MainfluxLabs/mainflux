// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, pkiAgent pki.Agent, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/certs", kithttp.NewServer(
		issueCert(svc),
		decodeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/certs/:id", kithttp.NewServer(
		viewCert(svc),
		decodeViewCert,
		encodeResponse,
		opts...,
	))

	r.Delete("/certs/:id", kithttp.NewServer(
		revokeCert(svc),
		decodeRevokeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/serials/:id", kithttp.NewServer(
		listSerials(svc),
		decodeListCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/crl", kithttp.NewServer(
		getCRL(svc),
		decodeGetCRL,
		encodeCRL,
		opts...,
	))

	r.Post("/certs/:id/renew", kithttp.NewServer(
		renewCert(svc),
		decodeViewCert,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("certs"))

	return r
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", apiutil.ContentTypeJSON)

	if ar, ok := response.(apiutil.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func decodeListCerts(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadUintQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}
	o, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeGetCRL(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func encodeCRL(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.WriteHeader(http.StatusOK)
	crl := response.([]byte)
	_, err := w.Write(crl)
	return err
}

func decodeViewCert(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewReq{
		token:    apiutil.ExtractBearerToken(r),
		serialID: bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != apiutil.ContentTypeJSON {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := addCertsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeRevokeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	req := revokeReq{
		token:  apiutil.ExtractBearerToken(r),
		certID: bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	apiutil.EncodeError(err, w)
	apiutil.WriteErrorResponse(err, w)
}

func verifyClientCertMiddleware(pkiAgent pki.Agent, logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				clientCert := r.TLS.PeerCertificates[0]

				certPEM := string(pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: clientCert.Raw,
				}))

				if _, err := pkiAgent.VerifyCert(certPEM); err != nil {
					logger.Warn(fmt.Sprintf("Client cert verification failed: %s", err))
					encodeError(r.Context(), err, w)
					return
				}

				ctx := context.WithValue(r.Context(), "client_cert", clientCert)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}
