// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package readers

import "context"

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListJSONMessages retrieves the json messages with given filters.
	ListJSONMessages(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// ListSenMLMessages retrieves the senml messages with given filters.
	ListSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// BackupJSONMessages backups the json messages with given filters.
	BackupJSONMessages(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error)

	// BackupSenMLMessages backups the senml messages with given filters.
	BackupSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error)

	// RestoreJSONMessages restores the json messages.
	RestoreJSONMessages(ctx context.Context, messages ...Message) error

	// RestoreSenMLMessages restores the senml messages.
	RestoreSenMLMessages(ctx context.Context, messages ...Message) error

	// DeleteJSONMessages deletes the json messages within a time range.
	DeleteJSONMessages(ctx context.Context, rpm JSONPageMetadata) error

	// DeleteSenMLMessages deletes the senml messages within a time range.
	DeleteSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) error
}

type readersService struct {
	json  JSONMessageRepository
	senml SenMLMessageRepository
}

func New(json JSONMessageRepository, senml SenMLMessageRepository) Service {
	return &readersService{
		json:  json,
		senml: senml,
	}
}

func (rs *readersService) ListJSONMessages(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	return rs.json.ListMessages(ctx, rpm)
}

func (rs *readersService) ListSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	return rs.senml.ListMessages(ctx, rpm)
}

func (rs *readersService) BackupJSONMessages(ctx context.Context, rpm JSONPageMetadata) (JSONMessagesPage, error) {
	return rs.json.Backup(ctx, rpm)
}

func (rs *readersService) BackupSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) (SenMLMessagesPage, error) {
	return rs.senml.Backup(ctx, rpm)
}

func (rs *readersService) RestoreJSONMessages(ctx context.Context, messages ...Message) error {
	return rs.json.Restore(ctx, messages...)
}

func (rs *readersService) RestoreSenMLMessages(ctx context.Context, messages ...Message) error {
	return rs.senml.Restore(ctx, messages...)
}

func (rs *readersService) DeleteJSONMessages(ctx context.Context, rpm JSONPageMetadata) error {
	return rs.json.DeleteMessages(ctx, rpm)
}

func (rs *readersService) DeleteSenMLMessages(ctx context.Context, rpm SenMLPageMetadata) error {
	return rs.senml.DeleteMessages(ctx, rpm)
}
