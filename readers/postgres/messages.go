// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package postgres

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/jmoiron/sqlx"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errTransRollback  = errors.New("failed to rollback transaction")
)

type postgresRepository struct {
	db              dbutil.Database
	jsonRepository  *jsonRepository
	senmlRepository *senmlRepository
}

func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db:              dbutil.NewDatabase(db),
		jsonRepository:  newJSONRepository(db),
		senmlRepository: newSenMLRepository(db),
	}
}

func (tr postgresRepository) ListJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return tr.jsonRepository.ListMessages(ctx, rpm)
}

func (tr postgresRepository) ListSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return tr.senmlRepository.ListMessages(ctx, rpm)
}

func (tr postgresRepository) BackupJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	return tr.jsonRepository.Backup(ctx, rpm)
}

func (tr postgresRepository) BackupSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return tr.senmlRepository.Backup(ctx, rpm)
}

func (tr postgresRepository) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	return tr.jsonRepository.Restore(ctx, messages...)
}

func (tr postgresRepository) RestoreSenMLMessages(ctx context.Context, messages ...readers.Message) error {
	return tr.senmlRepository.Restore(ctx, messages...)
}

func (tr postgresRepository) DeleteJSONMessages(ctx context.Context, rpm readers.JSONPageMetadata) error {
	return tr.jsonRepository.DeleteMessages(ctx, rpm)
}

func (tr postgresRepository) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	return tr.senmlRepository.DeleteMessages(ctx, rpm)
}
