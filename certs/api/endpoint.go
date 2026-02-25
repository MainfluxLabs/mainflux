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

		return issueCertRes{
			Certificate:    res.ClientCert,
			IssuingCA:      res.IssuingCA,
			CAChain:        res.CAChain,
			PrivateKey:     res.ClientKey,
			PrivateKeyType: res.PrivateKeyType,
			Serial:         res.Serial,
			ExpiresAt:      res.ExpiresAt,
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
				CertSerial: cert.Serial,
				ThingID:    cert.ThingID,
				ExpiresAt:  cert.ExpiresAt,
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

		return issueCertRes{
			Certificate:    cert.ClientCert,
			IssuingCA:      cert.IssuingCA,
			CAChain:        cert.CAChain,
			PrivateKey:     cert.ClientKey,
			PrivateKeyType: cert.PrivateKeyType,
			Serial:         cert.Serial,
			ExpiresAt:      cert.ExpiresAt,
		}, nil
	}
}
