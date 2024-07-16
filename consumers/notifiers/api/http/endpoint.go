// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createNotifiersEndpoint(svc notifiers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNotifiersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		nfs := []things.Notifier{}
		for _, nReq := range req.Notifiers {
			nf := things.Notifier{
				GroupID:  req.groupID,
				Contacts: nReq.Contacts,
			}
			nfs = append(nfs, nf)
		}

		saved, err := svc.CreateNotifiers(ctx, req.token, nfs...)
		if err != nil {
			return nil, err
		}

		return buildNotifiersResponse(saved, true), nil
	}
}

func listNotifiersByGroupEndpoint(svc notifiers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(notifierReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		nfs, err := svc.ListNotifiersByGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildNotifiersResponse(nfs, false), nil
	}
}

func viewNotifierEndpoint(svc notifiers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(notifierReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		notifier, err := svc.ViewNotifier(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildNotifierResponse(notifier, false), nil
	}
}

func updateNotifierEndpoint(svc notifiers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateNotifierReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		notifier := things.Notifier{
			ID:       req.id,
			Contacts: req.Contacts,
		}

		if err := svc.UpdateNotifier(ctx, req.token, notifier); err != nil {
			return nil, err
		}

		return notifierResponse{updated: true}, nil
	}
}

func removeNotifiersEndpoint(svc notifiers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeNotifiersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveNotifiers(ctx, req.token, req.groupID, req.NotifierIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildNotifiersResponse(notifiers []things.Notifier, created bool) notifiersRes {
	res := notifiersRes{Notifiers: []notifierResponse{}, created: created}
	for _, nf := range notifiers {
		notifier := notifierResponse{
			ID:       nf.ID,
			GroupID:  nf.GroupID,
			Contacts: nf.Contacts,
		}
		res.Notifiers = append(res.Notifiers, notifier)
	}

	return res
}

func buildNotifierResponse(notifier things.Notifier, updated bool) notifierResponse {
	res := notifierResponse{
		ID:       notifier.ID,
		GroupID:  notifier.GroupID,
		Contacts: notifier.Contacts,
		updated:  updated,
	}

	return res
}
