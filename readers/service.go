// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

const rootSubject = "root"

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListJSONMessages retrieves the json messages with given filters.
	ListJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// ListSenMLMessages retrieves the senml messages with given filters.
	ListSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// BackupJSONMessages backups the json messages with given filters.
	BackupJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// BackupSenMLMessages backups the senml messages with given filters.
	BackupSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// RestoreJSONMessages restores the json messages.
	RestoreJSONMessages(ctx context.Context, token string, messages ...Message) error

	// RestoreSenMLMessages restores the senml messages.
	RestoreSenMLMessages(ctx context.Context, token string, messages ...Message) error

	// DeleteJSONMessages deletes the json messages within a time range.
	DeleteJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) error

	// DeleteSenMLMessages deletes the senml messages within a time range.
	DeleteSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) error

	// DeleteJSONMessages deletes the json messages within a time range.
	DeleteAllJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) error

	// DeleteSenMLMessages deletes the senml messages within a time range.
	DeleteAllSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) error
}

type readersService struct {
	authc  protomfx.AuthServiceClient
	thingc protomfx.ThingsServiceClient
	json   JSONMessageRepository
	senml  SenMLMessageRepository
}

func New(auth protomfx.AuthServiceClient, things protomfx.ThingsServiceClient, json JSONMessageRepository, senml SenMLMessageRepository) Service {
	return &readersService{
		authc:  auth,
		thingc: things,
		json:   json,
		senml:  senml,
	}
}

func (rs *readersService) ListJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	switch {
	case rpm.Publisher != "":
		_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
		if err != nil {
			return JSONMessagesPage{}, err
		}
	case key.Value != "":
		pc, err := rs.getPubConfByKey(ctx, key)
		if err != nil {
			return JSONMessagesPage{}, err
		}
		rpm.Publisher = pc.PublisherID
	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return JSONMessagesPage{}, err
		}
	}

	return rs.json.Retrieve(ctx, rpm)
}

func (rs *readersService) ListSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	switch {
	case rpm.Publisher != "":
		_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
		if err != nil {
			return SenMLMessagesPage{}, err
		}
	case key.Value != "":
		pc, err := rs.getPubConfByKey(ctx, key)
		if err != nil {
			return SenMLMessagesPage{}, err
		}
		rpm.Publisher = pc.PublisherID
	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return SenMLMessagesPage{}, err
		}
	}

	return rs.senml.Retrieve(ctx, rpm)
}

func (rs *readersService) BackupJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	switch {
	case rpm.Publisher != "":
		_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
		if err != nil {
			return JSONMessagesPage{}, err
		}
	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return JSONMessagesPage{}, err
		}
	}

	return rs.json.Backup(ctx, rpm)
}

func (rs *readersService) BackupSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	switch {
	case rpm.Publisher != "":
		_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
		if err != nil {
			return SenMLMessagesPage{}, err
		}
	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return SenMLMessagesPage{}, err
		}
	}

	return rs.senml.Backup(ctx, rpm)
}

func (rs *readersService) RestoreJSONMessages(ctx context.Context, token string, messages ...Message) error {
	if err := rs.isAdmin(ctx, token); err != nil {
		return err
	}

	return rs.json.Restore(ctx, messages...)
}

func (rs *readersService) RestoreSenMLMessages(ctx context.Context, token string, messages ...Message) error {
	if err := rs.isAdmin(ctx, token); err != nil {
		return err
	}

	return rs.senml.Restore(ctx, messages...)
}

func (rs *readersService) DeleteJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) error {
	_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
	if err != nil {
		return err
	}

	return rs.json.Remove(ctx, rpm)
}

func (rs *readersService) DeleteSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) error {
	_, err := rs.thingc.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: rpm.Publisher, Action: auth.Viewer})
	if err != nil {
		return err
	}

	return rs.senml.Remove(ctx, rpm)
}

func (rs *readersService) DeleteAllJSONMessages(ctx context.Context, token string, rpm JSONPageMetadata) error {
	if err := rs.isAdmin(ctx, token); err != nil {
		return err
	}

	return rs.json.Remove(ctx, rpm)
}

func (rs *readersService) DeleteAllSenMLMessages(ctx context.Context, token string, rpm SenMLPageMetadata) error {
	if err := rs.isAdmin(ctx, token); err != nil {
		return err
	}

	return rs.senml.Remove(ctx, rpm)
}

func (rs *readersService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: rootSubject,
	}

	if _, err := rs.authc.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (rs *readersService) getPubConfByKey(ctx context.Context, key things.ThingKey) (*protomfx.PubConfByKeyRes, error) {
	pc, err := rs.thingc.GetPubConfByKey(ctx, &protomfx.ThingKey{Value: key.Value, Type: key.Type})
	if err != nil {
		return nil, err
	}

	return pc, nil
}
