// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"

	migrate "github.com/rubenv/sql-migrate"
)

// Config defines the options that are used when connecting to a PostgreSQL instance
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

// Connect creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate
// failure.
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
				Id: "ota_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS updates (
						id          UUID PRIMARY KEY,
						group_id    UUID NOT NULL,
						file_name   VARCHAR(255) NOT NULL,
						version     VARCHAR(254) NOT NULL,
						file_size   BIGINT NOT NULL DEFAULT 0,
						checksum    VARCHAR(64) NOT NULL DEFAULT '',
						description TEXT NOT NULL DEFAULT '',
						timeout     BIGINT NOT NULL,
						created_at  TIMESTAMPTZ NOT NULL
					)`,
					`CREATE INDEX IF NOT EXISTS idx_updates_group_id ON updates (group_id)`,
					`CREATE TABLE IF NOT EXISTS thing_updates (
						update_id        UUID NOT NULL REFERENCES updates (id) ON DELETE CASCADE,
						thing_id         UUID NOT NULL,
						status           VARCHAR(16) NOT NULL,
						error            VARCHAR(255) NOT NULL DEFAULT '',
						reported_version VARCHAR(254) NOT NULL DEFAULT '',
						updated_at       TIMESTAMPTZ NOT NULL,
						PRIMARY KEY (update_id, thing_id)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_thing_updates_thing_id_status ON thing_updates (thing_id, status)`,
					`CREATE INDEX IF NOT EXISTS idx_thing_updates_status_updated_at ON thing_updates (status, updated_at)`,
				},
				Down: []string{
					"DROP TABLE thing_updates",
					"DROP TABLE updates",
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
