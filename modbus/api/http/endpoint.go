// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/go-kit/kit/endpoint"
)

func createClientsEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createClientsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var cls []modbus.Client
		for _, dReq := range req.Clients {
			scheduler := cron.NormalizeTimezone(dReq.Scheduler)
			dataFields := toDataFields(dReq.DataFields)

			cl := modbus.Client{
				Name:         dReq.Name,
				IPAddress:    dReq.IPAddress,
				Port:         dReq.Port,
				SlaveID:      dReq.SlaveID,
				FunctionCode: dReq.FunctionCode,
				Scheduler:    scheduler,
				DataFields:   dataFields,
				Metadata:     dReq.Metadata,
			}
			cls = append(cls, cl)

		}

		saved, err := svc.CreateClients(ctx, req.token, req.thingID, cls...)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(saved, true), nil
	}
}

func listClientsByThingEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listClientsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cls, err := svc.ListClientsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildClientsPageResponse(cls), nil
	}
}

func listClientsByGroupEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listClientsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cls, err := svc.ListClientsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildClientsPageResponse(cls), nil
	}
}

func viewClientEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cl, err := svc.ViewClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildClientResponse(cl), nil
	}
}

func updateClientEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		scheduler := cron.NormalizeTimezone(req.Scheduler)
		dataFields := toDataFields(req.DataFields)

		cl := modbus.Client{
			ID:           req.id,
			Name:         req.Name,
			IPAddress:    req.IPAddress,
			Port:         req.Port,
			SlaveID:      req.SlaveID,
			FunctionCode: req.FunctionCode,
			Scheduler:    scheduler,
			DataFields:   dataFields,
			Metadata:     req.Metadata,
		}

		if err := svc.UpdateClient(ctx, req.token, cl); err != nil {
			return nil, err
		}

		return clientResponse{updated: true}, nil
	}
}

func removeClientsEndpoint(svc modbus.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeClientsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveClients(ctx, req.token, req.ClientIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func toDataFields(fields []field) []modbus.DataField {
	res := make([]modbus.DataField, len(fields))
	for i, f := range fields {
		res[i] = modbus.DataField{
			Name:      f.Name,
			Type:      f.Type,
			Unit:      f.Unit,
			Scale:     f.Scale,
			ByteOrder: f.ByteOrder,
			Address:   f.Address,
			Length:    f.Length,
		}
	}
	return res
}

func toDataFieldsRes(fields []modbus.DataField) []field {
	res := make([]field, len(fields))
	for i, f := range fields {
		res[i] = field{
			Name:      f.Name,
			Type:      f.Type,
			Unit:      f.Unit,
			Scale:     f.Scale,
			ByteOrder: f.ByteOrder,
			Address:   f.Address,
			Length:    f.Length,
		}
	}
	return res
}

func buildClientsResponse(cls []modbus.Client, created bool) clientsRes {
	res := clientsRes{Clients: []clientResponse{}, created: created}
	for _, md := range cls {
		dRes := buildClientResponse(md)
		res.Clients = append(res.Clients, dRes)
	}

	return res
}

func buildClientsPageResponse(cp modbus.ClientsPage) clientsPageRes {
	res := clientsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
			Order:  cp.Order,
			Dir:    cp.Dir,
			Name:   cp.Name,
		},
		Clients: []clientResponse{},
	}

	for _, md := range cp.Clients {
		dRes := buildClientResponse(md)
		res.Clients = append(res.Clients, dRes)
	}

	return res
}

func buildClientResponse(md modbus.Client) clientResponse {
	dataFields := toDataFieldsRes(md.DataFields)

	return clientResponse{
		ID:           md.ID,
		GroupID:      md.GroupID,
		ThingID:      md.ThingID,
		Name:         md.Name,
		IPAddress:    md.IPAddress,
		Port:         md.Port,
		SlaveID:      md.SlaveID,
		FunctionCode: md.FunctionCode,
		Scheduler:    md.Scheduler,
		DataFields:   dataFields,
		Metadata:     md.Metadata,
	}
}
