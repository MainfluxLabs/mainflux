// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

type Config struct {
	Host        string
	Port        string
	User        string
	Pass        string
	Name        string
	SSLMode     string
	SSLCert     string
	SSLKey      string
	SSLRootCert string
}

func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	if err := migrateDB(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "audit_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS events (
						id               UUID PRIMARY KEY,
						occurred_at      TIMESTAMPTZ NOT NULL,
						operation        VARCHAR(64) NOT NULL,
						actor_id         UUID,
						actor_email      VARCHAR(254),
						org_id           UUID,
						group_id         UUID,
						action_data      JSONB NOT NULL DEFAULT '{}'::jsonb
					)`,
					`CREATE INDEX IF NOT EXISTS events_occurred_at_idx ON events (occurred_at DESC)`,
					`CREATE INDEX IF NOT EXISTS events_actor_occurred_idx ON events (actor_id, occurred_at DESC)`,
					`CREATE INDEX IF NOT EXISTS events_op_occurred_idx ON events (operation, occurred_at DESC)`,
					`CREATE INDEX IF NOT EXISTS events_action_data_gin ON events USING GIN (action_data)`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS events_action_data_gin`,
					`DROP INDEX IF EXISTS events_op_occurred_idx`,
					`DROP INDEX IF EXISTS events_actor_occurred_idx`,
					`DROP INDEX IF EXISTS events_occurred_at_idx`,
					`DROP TABLE IF EXISTS events`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
