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
	retrieveAllCertificates      = "retrieve_all_certs"
	saveCert                     = "save_cert"
	removeCert                   = "remove_cert"
	retrieveRevokedCertificates  = "retrieve_revoked_certs"
	retrieveCertificatesByThing  = "retrieve_certs_by_thing_id"
	retrieveCertificatesBySerial = "retrieve_certs_by_serial"
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

func (crm certsRepositoryMiddleware) RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (certs.Page, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveAllCertificates)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx, ownerID, offset, limit)
}

func (crm certsRepositoryMiddleware) Save(ctx context.Context, cert certs.Cert) (string, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, saveCert)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, cert)
}

func (crm certsRepositoryMiddleware) Remove(ctx context.Context, ownerID, serial string) error {
	span := dbutil.CreateSpan(ctx, crm.tracer, removeCert)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, ownerID, serial)
}

func (crm certsRepositoryMiddleware) RetrieveRevokedCertificates(ctx context.Context) ([]certs.RevokedCert, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveRevokedCertificates)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveRevokedCerts(ctx)
}

func (crm certsRepositoryMiddleware) RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (certs.Page, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveCertificatesByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, ownerID, thingID, offset, limit)
}

func (crm certsRepositoryMiddleware) RetrieveBySerial(ctx context.Context, ownerID, serialID string) (certs.Cert, error) {
	span := dbutil.CreateSpan(ctx, crm.tracer, retrieveCertificatesBySerial)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveBySerial(ctx, ownerID, serialID)
}
