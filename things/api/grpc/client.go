// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout                time.Duration
	getPubConfigByKey      endpoint.Endpoint
	getConfigByThing       endpoint.Endpoint
	canUserAccessThing     endpoint.Endpoint
	canUserAccessProfile   endpoint.Endpoint
	canUserAccessGroup     endpoint.Endpoint
	canThingAccessGroup    endpoint.Endpoint
	identify               endpoint.Endpoint
	getGroupIDByThing      endpoint.Endpoint
	getGroupIDByProfile    endpoint.Endpoint
	getGroupIDsByOrg       endpoint.Endpoint
	getThingIDsByProfile   endpoint.Endpoint
	createGroupMemberships endpoint.Endpoint
	getGroup               endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.ThingsServiceClient {
	svcName := "protomfx.ThingsService"

	return &grpcClient{
		timeout: timeout,
		getPubConfigByKey: kitot.TraceClient(tracer, "get_pub_config_by_key")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetPubConfigByKey",
			encodeGetPubConfigByKeyRequest,
			decodeGetPubConfigByKeyResponse,
			protomfx.PubConfigByKeyRes{},
		).Endpoint()),
		getConfigByThing: kitot.TraceClient(tracer, "get_config_by_thing")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetConfigByThing",
			encodeGetConfigByThingRequest,
			decodeGetConfigByThingResponse,
			protomfx.ConfigByThingRes{},
		).Endpoint()),
		canUserAccessThing: kitot.TraceClient(tracer, "can_user_access_thing")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessThing",
			encodeUserAccessThingRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		canUserAccessProfile: kitot.TraceClient(tracer, "can_user_access_profile")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessProfile",
			encodeUserAccessProfileRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		canUserAccessGroup: kitot.TraceClient(tracer, "can_user_access_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessGroup",
			encodeUserAccessGroupRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		canThingAccessGroup: kitot.TraceClient(tracer, "can_thing_access_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanThingAccessGroup",
			encodeThingAccessGroupRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			protomfx.ThingID{},
		).Endpoint()),
		getGroupIDByThing: kitot.TraceClient(tracer, "get_group_id_by_thing")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupIDByThing",
			encodeGetGroupIDByThingRequest,
			decodeGetGroupIDResponse,
			protomfx.GroupID{},
		).Endpoint()),
		getGroupIDByProfile: kitot.TraceClient(tracer, "get_group_id_by_profile")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupIDByProfile",
			encodeGetGroupIDByProfileRequest,
			decodeGetGroupIDResponse,
			protomfx.GroupID{},
		).Endpoint()),
		getGroupIDsByOrg: kitot.TraceClient(tracer, "get_group_ids_by_org")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupIDsByOrg",
			encodeGetGroupIDsByOrgRequest,
			decodeGetGroupIDsResponse,
			protomfx.GroupIDs{},
		).Endpoint()),
		getThingIDsByProfile: kitot.TraceClient(tracer, "get_thing_ids_by_profile")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetThingIDsByProfile",
			encodeGetThingIDsByProfileRequest,
			decodeGetThingIDsResponse,
			protomfx.ThingIDs{},
		).Endpoint()),
		createGroupMemberships: kitot.TraceClient(tracer, "create_group_memebrships")(kitgrpc.NewClient(
			conn,
			svcName,
			"CreateGroupMemberships",
			encodeCreateGroupMembershipsRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		getGroup: kitot.TraceClient(tracer, "get_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroup",
			encodeGetGroupRequest,
			decodeGetGroupResponse,
			protomfx.Group{},
		).Endpoint()),
	}
}

func (client grpcClient) GetPubConfigByKey(ctx context.Context, req *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.PubConfigByKeyRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	tk := thingKey{
		value:   req.GetValue(),
		keyType: req.GetType(),
	}

	res, err := client.getPubConfigByKey(ctx, tk)
	if err != nil {
		return nil, err
	}

	pc := res.(pubConfigByKeyRes)
	return &protomfx.PubConfigByKeyRes{PublisherID: pc.publisherID, ProfileConfig: pc.profileConfig}, nil
}

func (client grpcClient) GetConfigByThing(ctx context.Context, req *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ConfigByThingRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	res, err := client.getConfigByThing(ctx, thingIDReq{thingID: req.GetValue()})
	if err != nil {
		return nil, err
	}
	c := res.(configByThingRes)
	return &protomfx.ConfigByThingRes{Config: c.config}, nil
}

func (client grpcClient) CanUserAccessThing(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessThingReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessThing(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func (client grpcClient) CanUserAccessProfile(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessProfileReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessProfile(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func (client grpcClient) CanUserAccessGroup(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessGroupReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessGroup(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func (client grpcClient) CanThingAccessGroup(ctx context.Context, req *protomfx.ThingAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := thingAccessGroupReq{thingKey: thingKey{value: req.GetKey()}, id: req.GetId()}
	res, err := client.canThingAccessGroup(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func (client grpcClient) Identify(ctx context.Context, req *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, thingKey{value: req.GetValue(), keyType: req.GetType()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &protomfx.ThingID{Value: ir.id}, nil
}

func (client grpcClient) GetGroupIDByThing(ctx context.Context, req *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupIDByThing(ctx, thingIDReq{thingID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	tg := res.(groupIDRes)
	return &protomfx.GroupID{Value: tg.groupID}, nil
}

func (client grpcClient) GetGroupIDByProfile(ctx context.Context, req *protomfx.ProfileID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupIDByProfile(ctx, profileIDReq{profileID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	pg := res.(groupIDRes)
	return &protomfx.GroupID{Value: pg.groupID}, nil
}

func (client grpcClient) GetGroupIDsByOrg(ctx context.Context, req *protomfx.OrgAccessReq, _ ...grpc.CallOption) (*protomfx.GroupIDs, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupIDsByOrg(ctx, orgAccessReq{orgID: req.GetOrgId(), token: req.GetToken()})
	if err != nil {
		return nil, err
	}

	ids := res.(groupIDsRes)
	return &protomfx.GroupIDs{Ids: ids.groupIDs}, nil
}

func (client grpcClient) GetThingIDsByProfile(ctx context.Context, req *protomfx.ProfileID, _ ...grpc.CallOption) (*protomfx.ThingIDs, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getThingIDsByProfile(ctx, profileIDReq{profileID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	ids := res.(thingIDsRes)
	return &protomfx.ThingIDs{Ids: ids.thingIDs}, nil
}

func (client grpcClient) CreateGroupMemberships(ctx context.Context, req *protomfx.CreateGroupMembershipsReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	memberships := req.GetMemberships()

	clientReq := createGroupMembershipsReq{
		memberships: make([]groupMembership, 0, len(memberships)),
	}

	for _, memb := range memberships {
		clientReq.memberships = append(clientReq.memberships, groupMembership{
			userID:  memb.GetUserID(),
			groupID: memb.GetGroupID(),
			role:    memb.GetRole(),
		})
	}

	if _, err := client.createGroupMemberships(ctx, clientReq); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (client grpcClient) GetGroup(ctx context.Context, req *protomfx.GetGroupReq, _ ...grpc.CallOption) (*protomfx.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroup(ctx, getGroupReq{groupID: req.GetGroupID()})
	if err != nil {
		return nil, err
	}

	gr := res.(groupRes)
	return &protomfx.Group{
		Id:    gr.id,
		OrgID: gr.orgID,
		Name:  gr.name,
	}, nil
}

func encodeGetPubConfigByKeyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(thingKey)
	return &protomfx.ThingKey{Value: req.value, Type: req.keyType}, nil
}

func encodeGetConfigByThingRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(thingIDReq)
	return &protomfx.ThingID{Value: req.thingID}, nil
}

func encodeUserAccessThingRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(userAccessThingReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeUserAccessProfileRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(userAccessProfileReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeUserAccessGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(userAccessGroupReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeThingAccessGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(thingAccessGroupReq)
	return &protomfx.ThingAccessReq{Key: req.thingKey.value, Id: req.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(thingKey)
	return &protomfx.ThingKey{Value: req.value, Type: req.keyType}, nil
}

func encodeGetGroupIDByThingRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(thingIDReq)
	return &protomfx.ThingID{Value: req.thingID}, nil
}

func encodeGetGroupIDByProfileRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(profileIDReq)
	return &protomfx.ProfileID{Value: req.profileID}, nil
}

func encodeGetGroupIDsByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(orgAccessReq)
	return &protomfx.OrgAccessReq{
		OrgId: req.orgID,
		Token: req.token,
	}, nil
}

func encodeGetThingIDsByProfileRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(profileIDReq)
	return &protomfx.ProfileID{
		Value: req.profileID,
	}, nil
}

func encodeCreateGroupMembershipsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(createGroupMembershipsReq)

	protoReq := &protomfx.CreateGroupMembershipsReq{
		Memberships: make([]*protomfx.GroupMembership, 0, len(req.memberships)),
	}

	for _, memb := range req.memberships {
		protoReq.Memberships = append(protoReq.Memberships, &protomfx.GroupMembership{
			UserID:  memb.userID,
			GroupID: memb.groupID,
			Role:    memb.role,
		})
	}

	return protoReq, nil
}

func encodeGetGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(getGroupReq)
	return &protomfx.GetGroupReq{GroupID: req.groupID}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.ThingID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeGetPubConfigByKeyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.PubConfigByKeyRes)
	return pubConfigByKeyRes{publisherID: res.PublisherID, profileConfig: res.ProfileConfig}, nil
}

func decodeGetConfigByThingResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.ConfigByThingRes)
	return configByThingRes{config: res.GetConfig()}, nil
}

func decodeEmptyResponse(_ context.Context, _ any) (any, error) {
	return emptyRes{}, nil
}

func decodeGetGroupIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.GroupID)
	return groupIDRes{groupID: res.GetValue()}, nil
}

func decodeGetGroupIDsResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.GroupIDs)
	return groupIDsRes{groupIDs: res.GetIds()}, nil
}

func decodeGetThingIDsResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.ThingIDs)
	return thingIDsRes{thingIDs: res.GetIds()}, nil
}

func decodeGetGroupResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.Group)
	return groupRes{
		id:    res.GetId(),
		orgID: res.GetOrgID(),
		name:  res.GetName(),
	}, nil
}
