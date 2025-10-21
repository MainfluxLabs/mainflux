// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/go-kit/kit/endpoint"
)

func issueCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addCertsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res, err := svc.IssueCert(ctx, req.token, req.ThingID, req.TTL, req.KeyBits, req.KeyType)
		if err != nil {
			return certsRes{}, err
		}

		return certsRes{
			CertSerial: res.Serial,
			ThingID:    res.ThingID,
			ClientCert: res.ClientCert,
			ClientKey:  res.ClientKey,
			Expiration: res.Expire,
			created:    true,
		}, nil
	}
}

func listSerials(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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
			}
			res.Certs = append(res.Certs, cr)
		}
		return res, nil
	}
}

func viewCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.ViewCert(ctx, req.token, req.serialID)
		if err != nil {
			return certsPageRes{}, err
		}

		certRes := certsRes{
			CertSerial: cert.Serial,
			ThingID:    cert.ThingID,
			ClientCert: cert.ClientCert,
			Expiration: cert.Expire,
		}

		return certRes, nil
	}
}

func revokeCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(revokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		return svc.RevokeCert(ctx, req.token, req.certID)
	}
}

func getCRL(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		crl, err := svc.GetCRL(ctx)
		if err != nil {
			return nil, err
		}
		return crl, nil
	}
}

func renewCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cert, err := svc.RenewCert(ctx, req.token, req.serialID)
		if err != nil {
			return certsRes{}, err
		}

		return certsRes{
			CertSerial: cert.Serial,
			ThingID:    cert.ThingID,
			ClientCert: cert.ClientCert,
			ClientKey:  cert.ClientKey,
			Expiration: cert.Expire,
			created:    true,
		}, nil
	}
}
