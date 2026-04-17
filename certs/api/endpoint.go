// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/go-kit/kit/endpoint"
)

func issueCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(addCertsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res, err := svc.IssueCert(ctx, req.token, req.ThingID, req.TTL, req.KeyBits, req.KeyType)
		if err != nil {
			return issueCertRes{}, err
		}

		return buildIssueCertRes(res), nil
	}
}

func rotateCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(rotateCertsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		res, err := svc.RotateCert(ctx, req.token, req.serial, req.ThingID, req.TTL, req.KeyBits, req.KeyType)
		if err != nil {
			return nil, err
		}

		return buildIssueCertRes(res), nil
	}
}

func listSerialsByThingEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListSerials(ctx, req.token, req.thingID, req.offset, req.limit)
		if err != nil {
			return certsPageRes{}, err
		}
		res := certsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: req.offset,
				Limit:  req.limit,
			},
			Certs: []certsRes{},
		}

		for _, cert := range page.Certs {
			cr := certsRes{
				CertSerial: cert.Serial,
				ThingID:    cert.ThingID,
				ExpiresAt:  cert.ExpiresAt,
				Downloaded: cert.Downloaded,
			}
			res.Certs = append(res.Certs, cr)
		}
		return res, nil
	}
}

func viewCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.ViewCert(ctx, req.token, req.serial)
		if err != nil {
			return viewCertRes{}, err
		}

		return viewCertRes{
			Certificate: cert.ClientCert,
			Serial:      cert.Serial,
			ExpiresAt:   cert.ExpiresAt,
			ThingID:     cert.ThingID,
			Downloaded:  cert.Downloaded,
			KeyType:     cert.PrivateKeyType,
			KeyBits:     cert.KeyBits,
		}, nil
	}
}

func revokeCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(revokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		return svc.RevokeCert(ctx, req.token, req.serial)
	}
}

func downloadCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.DownloadCert(ctx, req.token, req.serial)
		if err != nil {
			return downloadCertRes{}, err
		}

		return downloadCertRes{
			Certificate:    cert.ClientCert,
			IssuingCA:      cert.IssuingCA,
			CAChain:        cert.CAChain,
			PrivateKey:     cert.ClientKey,
			PrivateKeyType: cert.PrivateKeyType,
			Serial:         cert.Serial,
			ThingID:        cert.ThingID,
			ExpiresAt:      cert.ExpiresAt,
		}, nil
	}
}

func renewCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.RenewCert(ctx, req.token, req.serial)
		if err != nil {
			return issueCertRes{}, err
		}

		return buildIssueCertRes(cert), nil
	}
}

func buildIssueCertRes(c certs.Cert) issueCertRes {
	return issueCertRes{
		Certificate:    c.ClientCert,
		IssuingCA:      c.IssuingCA,
		CAChain:        c.CAChain,
		PrivateKey:     c.ClientKey,
		PrivateKeyType: c.PrivateKeyType,
		Serial:         c.Serial,
		ExpiresAt:      c.ExpiresAt,
	}
}
