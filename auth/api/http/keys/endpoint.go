// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(issueKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		newKey := auth.Key{
			IssuedAt: now,
			Type:     req.Type,
		}

		duration := time.Duration(req.Duration * time.Second)
		if duration != 0 {
			exp := now.Add(duration)
			newKey.ExpiresAt = exp
		}

		key, secret, err := svc.Issue(ctx, req.token, newKey)
		if err != nil {
			return nil, err
		}

		res := issueKeyRes{
			ID:       key.ID,
			Value:    secret,
			IssuedAt: key.IssuedAt,
		}
		if !key.ExpiresAt.IsZero() {
			res.ExpiresAt = &key.ExpiresAt
		}
		return res, nil
	}
}

func retrieveEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		key, err := svc.RetrieveKey(ctx, req.token, req.id)

		if err != nil {
			return nil, err
		}
		ret := retrieveKeyRes{
			ID:       key.ID,
			IssuerID: key.IssuerID,
			Subject:  key.Subject,
			Type:     key.Type,
			IssuedAt: key.IssuedAt,
		}
		if !key.ExpiresAt.IsZero() {
			ret.ExpiresAt = &key.ExpiresAt
		}

		return ret, nil
	}
}

func revokeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Revoke(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return revokeKeyRes{}, nil
	}
}

func listAPIKeysEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listKeysReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAPIKeys(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildKeysResponse(page, req.pageMetadata), nil
	}
}

func buildKeysResponse(kp auth.KeysPage, pm auth.PageMetadata) keysPageRes {
	res := keysPageRes{
		pageRes: pageRes{
			Total:  kp.Total,
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		Keys: []retrieveKeyRes{},
	}

	for _, k := range kp.Keys {
		view := retrieveKeyRes{
			ID:       k.ID,
			IssuerID: k.IssuerID,
			Subject:  k.Subject,
			Type:     k.Type,
			IssuedAt: k.IssuedAt,
		}
		if !k.ExpiresAt.IsZero() {
			view.ExpiresAt = &k.ExpiresAt
		}
		res.Keys = append(res.Keys, view)
	}

	return res
}
