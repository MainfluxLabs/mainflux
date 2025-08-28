// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package postgres

import (
	"context"
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
	db              *sqlx.DB
	jsonRepository  *jsonRepository
	senmlRepository *senmlRepository
	aggregator      *aggregationService
}

func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db:              db,
		jsonRepository:  newJSONRepository(db),
		senmlRepository: newSenMLRepository(db),
	}
}

func (tr postgresRepository) ListJSONMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.jsonRepository.ListMessages(rpm)
}

func (tr postgresRepository) ListSenMLMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.senmlRepository.ListMessages(rpm)
}

func (tr postgresRepository) BackupJSONMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.jsonRepository.Backup(rpm)
}

func (tr postgresRepository) BackupSenMLMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return tr.senmlRepository.Backup(rpm)
}

func (tr postgresRepository) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	return tr.jsonRepository.Restore(ctx, messages...)
}

func (tr postgresRepository) RestoreSenMLMessageS(ctx context.Context, messages ...readers.Message) error {
	return tr.senmlRepository.Restore(ctx, messages...)
}

func (tr postgresRepository) DeleteJSONMessages(ctx context.Context, rpm readers.PageMetadata) error {
	return tr.jsonRepository.DeleteMessages(ctx, rpm)
}

func (tr postgresRepository) DeleteSenMLMessages(ctx context.Context, rpm readers.PageMetadata) error {
	return tr.senmlRepository.DeleteMessages(ctx, rpm)
}
