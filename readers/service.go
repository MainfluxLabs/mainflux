// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	jsonFormat  = "json"
	senmlFormat = "senml"
	csvFormat   = "csv"
	rootSubject = "root"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListJSONMessages retrieves the json messages with given filters.
	ListJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// ListSenMLMessages retrieves the senml messages with given filters.
	ListSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// BackupJSONMessages backups the json messages with given filters.
	BackupJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// BackupSenMLMessages backups the senml messages with given filters.
	BackupSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// RestoreJSONMessages restores the json messages.
	RestoreJSONMessages(ctx context.Context, token string, messages ...Message) error

	// RestoreSenMLMessages restores the senml messages.
	RestoreSenMLMessages(ctx context.Context, token string, messages ...Message) error

	// DeleteJSONMessages deletes the json messages within a time range.
	DeleteJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) error

	// DeleteSenMLMessages deletes the senml messages within a time range.
	DeleteSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) error
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

func (rs *readersService) ListJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	switch {
	case key != "":
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

func (rs *readersService) ListSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	switch {
	case key != "":
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

func (rs *readersService) BackupJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	switch {
	case key != "":
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

	return rs.json.Backup(ctx, rpm)
}

func (rs *readersService) BackupSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	switch {
	case key != "":
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

func (rs *readersService) DeleteJSONMessages(ctx context.Context, token, key string, rpm JSONPageMetadata) error {
	switch {
	case key != "":
		pc, err := rs.getPubConfByKey(ctx, key)
		if err != nil {
			return errors.Wrap(errors.ErrAuthentication, err)
		}
		rpm.Publisher = pc.PublisherID

	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return err
		}
	}

	return rs.json.Remove(ctx, rpm)
}

func (rs *readersService) DeleteSenMLMessages(ctx context.Context, token, key string, rpm SenMLPageMetadata) error {
	switch {
	case key != "":
		pc, err := rs.getPubConfByKey(ctx, key)
		if err != nil {
			return errors.Wrap(errors.ErrAuthentication, err)
		}
		rpm.Publisher = pc.PublisherID

	default:
		if err := rs.isAdmin(ctx, token); err != nil {
			return err
		}
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

func (rs *readersService) getPubConfByKey(ctx context.Context, key string) (*protomfx.PubConfByKeyRes, error) {
	pc, err := rs.thingc.GetPubConfByKey(ctx, &protomfx.PubConfByKeyReq{Key: key})
	if err != nil {
		return nil, err
	}

	return pc, nil
}
