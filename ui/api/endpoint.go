// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/ui"
)

func indexEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(indexReq)
		res, err := svc.Index(ctx, req.token)
		return uiRes{
			html: res,
		}, err
	}
}

func createThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createThingsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		th := sdk.Thing{
			Key:      req.Key,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		res, err := svc.CreateThings(ctx, req.token, th)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}

func viewThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ViewThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func listThingsEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listThingsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ListThings(ctx, req.token)
		return uiRes{
			html: res,
		}, err
	}
}

func updateThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		uth := sdk.Thing{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		res, err := svc.UpdateThing(ctx, req.token, req.id, uth)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func removeThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.RemoveThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html:    res,
			headers: map[string]string{"location": redirectURL + "things"},
			code:    http.StatusPermanentRedirect,
		}, err
	}
}

func createChannelEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		ch := sdk.Channel{
			Key:      req.Key,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		res, err := svc.CreateChannels(ctx, req.token, ch)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}

func viewChannelEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ViewChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func updateChannelEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		uch := sdk.Channel{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		res, err := svc.UpdateChannel(ctx, req.token, req.id, uch)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func listChannelsEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listChannelsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ListChannels(ctx, req.token)
		return uiRes{
			html: res,
		}, err
	}
}

func removeChannelEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.RemoveChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html:    res,
			headers: map[string]string{"location": redirectURL + "channels"},
			code:    http.StatusPermanentRedirect,
		}, err
	}
}

func connectThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectThingReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		res, err := svc.Connect(ctx, cr.token, []string{cr.ChanID}, []string{cr.ThingID})
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func listThingConnectionsEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ListThingConnections(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func listChannelConnectionsEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ListChannelConnections(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func disconnectThingEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		dcr := request.(disconnectThingReq)

		if err := dcr.validate(); err != nil {
			return nil, err
		}

		res, err := svc.DisconnectThing(ctx, dcr.token, []string{dcr.ChanID}, []string{dcr.ThingID})
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func disconnectChannelEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		dcr := request.(disconnectChannelReq)

		if err := dcr.validate(); err != nil {
			return nil, err
		}

		res, err := svc.DisconnectChannel(ctx, dcr.token, []string{dcr.ThingID}, []string{dcr.ChanID})
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func createGroupEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		gr := sdk.Group{
			Name:        req.Name,
			Description: req.Description,
			ParentID:    req.ParentID,
			Metadata:    req.Metadata,
		}

		res, err := svc.CreateGroups(ctx, req.token, gr)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}

func listGroupsEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ListGroups(ctx, req.token)
		return uiRes{
			html: res,
		}, err
	}
}

func viewGroupEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.ViewGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}

func assignEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		res, err := svc.Assign(ctx, req.token, req.groupID, req.Type, req.Member)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}

func unassignEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		res, err := svc.Unassign(ctx, req.token, req.groupID, req.Member)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func updateGroupEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		ugr := sdk.Group{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		res, err := svc.UpdateGroup(ctx, req.token, req.id, ugr)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func removeGroupEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }

		res, err := svc.RemoveGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html:    res,
			headers: map[string]string{"location": redirectURL + "groups"},
			code:    http.StatusPermanentRedirect,
		}, err
	}
}

func sendMessageEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(sendMessageReq)
		// if err := req.validate(); err != nil {
		// 	return nil, err
		// }
		res, err := svc.SendMessage(ctx, req.token)
		if err != nil {
			return nil, err
		}
		return uiRes{
			html: res,
		}, err
	}
}

func publishMessageEndpoint(svc ui.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(publishReq)
		res, err := svc.Publish(ctx, req.thingKey, req.msg)
		if err != nil {
			return nil, err
		}

		return uiRes{
			html: res,
		}, err
	}
}
