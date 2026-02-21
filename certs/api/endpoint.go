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
			return certsRes{}, err
		}

		return certsRes{
			CertSerial:     res.Serial,
			ThingID:        res.ThingID,
			ClientCert:     res.ClientCert,
			ClientKey:      res.ClientKey,
			ExpiresAt:      res.ExpiresAt,
			PrivateKeyType: res.PrivateKeyType,
			KeyBits:        res.KeyBits,
			created:        true,
		}, nil
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
				CertSerial:     cert.Serial,
				ThingID:        cert.ThingID,
				ExpiresAt:      cert.ExpiresAt,
				PrivateKeyType: cert.PrivateKeyType,
				KeyBits:        cert.KeyBits,
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
			return certsPageRes{}, err
		}

		certRes := certsRes{
			CertSerial:     cert.Serial,
			ThingID:        cert.ThingID,
			ClientCert:     cert.ClientCert,
			ExpiresAt:      cert.ExpiresAt,
			PrivateKeyType: cert.PrivateKeyType,
			KeyBits:        cert.KeyBits,
		}

		return certRes, nil
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

func renewCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.RenewCert(ctx, req.token, req.serial)
		if err != nil {
			return certsRes{}, err
		}

		return certsRes{
			CertSerial:     cert.Serial,
			ThingID:        cert.ThingID,
			ClientCert:     cert.ClientCert,
			ClientKey:      cert.ClientKey,
			ExpiresAt:      cert.ExpiresAt,
			PrivateKeyType: cert.PrivateKeyType,
			KeyBits:        cert.KeyBits,
			created:        true,
		}, nil
	}
}
