// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	retrieveAllCerts     = "retrieve_all_certs"
	saveCert             = "save_cert"
	removeCert           = "remove_cert"
	retrieveRevokedCerts = "retrieve_revoked_certs"
	retrieveCertByThing  = "retrieve_cert_by_thing"
	retrieveCertBySerial = "retrieve_cert_by_serial"
)

var _ certs.Repository = (*certsRepositoryMiddleware)(nil)

type certsRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   certs.Repository
}

func CertsRepositoryMiddleware(tracer opentracing.Tracer, repo certs.Repository) certs.Repository {
	return certsRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm certsRepositoryMiddleware) RetrieveAll(ctx context.Context, offset, limit uint64) (certs.Page, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveAllCerts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx, offset, limit)
}

func (crm certsRepositoryMiddleware) Save(ctx context.Context, cert certs.Cert) (string, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, saveCert)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, cert)
}

func (crm certsRepositoryMiddleware) Remove(ctx context.Context, serialID string) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, removeCert)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, serialID)
}

func (crm certsRepositoryMiddleware) RetrieveRevokedCerts(ctx context.Context) ([]certs.RevokedCert, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveRevokedCerts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveRevokedCerts(ctx)
}

func (crm certsRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, offset, limit uint64) (certs.Page, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveCertByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, thingID, offset, limit)
}

func (crm certsRepositoryMiddleware) RetrieveBySerial(ctx context.Context, serialID string) (certs.Cert, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveCertBySerial)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveBySerial(ctx, serialID)
}
