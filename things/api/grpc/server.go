// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	getPubConfByKey        kitgrpc.Handler
	getConfigByThingID     kitgrpc.Handler
	canUserAccessThing     kitgrpc.Handler
	canUserAccessProfile   kitgrpc.Handler
	canUserAccessGroup     kitgrpc.Handler
	canThingAccessGroup    kitgrpc.Handler
	identify               kitgrpc.Handler
	getGroupIDByThingID    kitgrpc.Handler
	getGroupIDByProfileID  kitgrpc.Handler
	getGroupIDsByOrg       kitgrpc.Handler
	getThingIDsByProfile   kitgrpc.Handler
	createGroupMemberships kitgrpc.Handler
	getGroup               kitgrpc.Handler
	getKeyByThingID        kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc things.Service) protomfx.ThingsServiceServer {
	return &grpcServer{
		getPubConfByKey: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_pub_conf_by_key")(getPubConfByKeyEndpoint(svc)),
			decodeGetPubConfByKeyRequest,
			encodeGetPubConfByKeyResponse,
		),
		getConfigByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_config_by_thing_id")(getConfigByThingIDEndpoint(svc)),
			decodeGetConfigByThingIDRequest,
			encodeGetConfigByThingIDResponse,
		),
		canUserAccessThing: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_thing")(canUserAccessThingEndpoint(svc)),
			decodeUserAccessThingRequest,
			encodeEmptyResponse,
		),
		canUserAccessProfile: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_profile")(canUserAccessProfileEndpoint(svc)),
			decodeUserAccessProfileRequest,
			encodeEmptyResponse,
		),
		canUserAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_group")(canUserAccessGroupEndpoint(svc)),
			decodeUserAccessGroupRequest,
			encodeEmptyResponse,
		),
		canThingAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_thing_access_group")(canThingAccessGroupEndpoint(svc)),
			decodeThingAccessGroupRequest,
			encodeEmptyResponse,
		),
		identify: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentityResponse,
		),
		getGroupIDByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_id_by_thing_id")(getGroupIDByThingIDEndpoint(svc)),
			decodeGetGroupIDByThingIDRequest,
			encodeGetGroupIDByThingIDResponse,
		),
		getGroupIDByProfileID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_id_by_profile_id")(getGroupIDByProfileIDEndpoint(svc)),
			decodeGetGroupIDByProfileIDRequest,
			encodeGetGroupIDByProfileIDResponse,
		),
		getGroupIDsByOrg: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_ids_by_org")(getGroupIDsByOrgEndpoint(svc)),
			decodeGetGroupIDsByOrgRequest,
			encodeGetGroupIDsByOrgResponse,
		),
		getThingIDsByProfile: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_thing_ids_by_profile")(getThingIDsByProfileEndpoint(svc)),
			decodeGetThingIDsByProfileRequest,
			encodeGetThingIDsByProfileResponse,
		),
		createGroupMemberships: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "create_group_memberships")(createGroupMembershipsEndpoint(svc)),
			decodeCreateGroupMembershipsRequest,
			encodeEmptyResponse,
		),
		getGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group")(getGroupEndpoint(svc)),
			decodeGetGroupRequest,
			encodeGetGroupResponse,
		),
		getKeyByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_key_by_thing_id")(getKeyByThingIDEndpoint(svc)),
			decodeGetKeyByThingIDRequest,
			encodeGetKeyByThingIDResponse,
		),
	}
}

func (gs *grpcServer) GetPubConfByKey(ctx context.Context, req *protomfx.ThingKey) (*protomfx.PubConfByKeyRes, error) {
	_, res, err := gs.getPubConfByKey.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.PubConfByKeyRes), nil
}

func (gs *grpcServer) GetConfigByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.ConfigByThingIDRes, error) {
	_, res, err := gs.getConfigByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.ConfigByThingIDRes), nil
}

func (gs *grpcServer) CanUserAccessThing(ctx context.Context, req *protomfx.UserAccessReq) (*emptypb.Empty, error) {
	_, res, err := gs.canUserAccessThing.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (gs *grpcServer) CanUserAccessProfile(ctx context.Context, req *protomfx.UserAccessReq) (*emptypb.Empty, error) {
	_, res, err := gs.canUserAccessProfile.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (gs *grpcServer) CanUserAccessGroup(ctx context.Context, req *protomfx.UserAccessReq) (*emptypb.Empty, error) {
	_, res, err := gs.canUserAccessGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (gs *grpcServer) CanThingAccessGroup(ctx context.Context, req *protomfx.ThingAccessReq) (*emptypb.Empty, error) {
	_, res, err := gs.canThingAccessGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (gs *grpcServer) Identify(ctx context.Context, req *protomfx.ThingKey) (*protomfx.ThingID, error) {
	_, res, err := gs.identify.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.ThingID), nil
}

func (gs *grpcServer) GetGroupIDByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.GroupID, error) {
	_, res, err := gs.getGroupIDByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.GroupID), nil
}

func (gs *grpcServer) GetGroupIDByProfileID(ctx context.Context, req *protomfx.ProfileID) (*protomfx.GroupID, error) {
	_, res, err := gs.getGroupIDByProfileID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.GroupID), nil
}

func (gs *grpcServer) GetGroupIDsByOrg(ctx context.Context, req *protomfx.OrgAccessReq) (*protomfx.GroupIDs, error) {
	_, res, err := gs.getGroupIDsByOrg.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.GroupIDs), nil
}

func (gs *grpcServer) GetThingIDsByProfile(ctx context.Context, req *protomfx.ProfileID) (*protomfx.ThingIDs, error) {
	_, res, err := gs.getThingIDsByProfile.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.ThingIDs), nil
}

func (gs *grpcServer) CreateGroupMemberships(ctx context.Context, req *protomfx.CreateGroupMembershipsReq) (*emptypb.Empty, error) {
	_, res, err := gs.createGroupMemberships.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (gs *grpcServer) GetGroup(ctx context.Context, req *protomfx.GetGroupReq) (*protomfx.Group, error) {
	_, res, err := gs.getGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.Group), nil
}

func (gs *grpcServer) GetKeyByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.ThingKeyRes, error) {
	_, res, err := gs.getKeyByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.ThingKeyRes), nil
}

func decodeGetPubConfByKeyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingKey)
	return thingKey{value: req.GetValue(), keyType: req.GetType()}, nil
}

func decodeGetConfigByThingIDRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingID)
	return thingIDReq{thingID: req.GetValue()}, nil
}

func decodeUserAccessThingRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessThingReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeUserAccessProfileRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessProfileReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeUserAccessGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessGroupReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeThingAccessGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingAccessReq)
	return thingAccessGroupReq{thingKey: thingKey{value: req.GetKey()}, id: req.GetId()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingKey)
	return thingKey{value: req.GetValue(), keyType: req.GetType()}, nil
}

func decodeGetGroupIDByThingIDRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingID)
	return thingIDReq{thingID: req.GetValue()}, nil
}

func decodeGetGroupIDByProfileIDRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ProfileID)
	return profileIDReq{profileID: req.GetValue()}, nil
}

func decodeGetGroupIDsByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.OrgAccessReq)
	return orgAccessReq{orgID: req.GetOrgId(), token: req.GetToken()}, nil
}

func decodeGetThingIDsByProfileRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ProfileID)
	return profileIDReq{profileID: req.GetValue()}, nil
}

func decodeCreateGroupMembershipsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.CreateGroupMembershipsReq)
	memberships := req.GetMemberships()

	ret := createGroupMembershipsReq{
		memberships: make([]groupMembership, 0, len(memberships)),
	}

	for _, membership := range memberships {
		ret.memberships = append(ret.memberships, groupMembership{
			userID:  membership.GetUserID(),
			groupID: membership.GetGroupID(),
			role:    membership.GetRole(),
		})
	}

	return ret, nil
}

func decodeGetGroupRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.GetGroupReq)
	return getGroupReq{groupID: req.GetGroupID()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(identityRes)
	return &protomfx.ThingID{Value: res.id}, nil
}

func encodeGetPubConfByKeyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(pubConfByKeyRes)
	return &protomfx.PubConfByKeyRes{PublisherID: res.publisherID, ProfileConfig: res.profileConfig}, nil
}

func encodeGetConfigByThingIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(configByThingIDRes)
	return &protomfx.ConfigByThingIDRes{Config: res.config}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(emptyRes)
	return &emptypb.Empty{}, encodeError(res.err)
}

func encodeGetGroupIDByThingIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(groupIDRes)
	return &protomfx.GroupID{Value: res.groupID}, nil
}

func encodeGetGroupIDByProfileIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(groupIDRes)
	return &protomfx.GroupID{Value: res.groupID}, nil
}

func encodeGetGroupIDsByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(groupIDsRes)
	return &protomfx.GroupIDs{Ids: res.groupIDs}, nil
}

func encodeGetThingIDsByProfileResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(thingIDsRes)
	return &protomfx.ThingIDs{Ids: res.thingIDs}, nil
}

func encodeGetGroupResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(groupRes)

	return &protomfx.Group{
		Id:    res.id,
		OrgID: res.orgID,
		Name:  res.name,
	}, nil
}

func decodeGetKeyByThingIDRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ThingID)
	return thingIDReq{thingID: req.GetValue()}, nil
}

func encodeGetKeyByThingIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(thingKeyRes)
	return &protomfx.ThingKeyRes{Value: res.key}, nil
}

func encodeError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case err == nil:
		return nil
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingThingID,
		err == apiutil.ErrMissingProfileID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrInvalidAction,
		err == apiutil.ErrBearerToken,
		err == apiutil.ErrBearerKey,
		err == apiutil.ErrEmptyList:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, dbutil.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
