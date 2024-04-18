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
				Id: "things_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS things (
						id        UUID UNIQUE NOT NULL,
						owner     UUID NOT NULL,
						key       VARCHAR(4096) UNIQUE NOT NULL,
						name      VARCHAR(1024),
						metadata  JSONB,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						id           UUID UNIQUE NOT NULL,
						owner        UUID NOT NULL,
						name         VARCHAR(1024),
						profile      JSONB,
						metadata     JSONB,
						PRIMARY      KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    UUID,
						thing_id      UUID,
						FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id) REFERENCES things (id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, thing_id)
					)`,
					`CREATE TABLE IF NOT EXISTS groups (
						id          UUID UNIQUE NOT NULL,
						owner_id    UUID NOT NULL,
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						PRIMARY KEY (id, owner_id)
					)`,
					`CREATE TABLE IF NOT EXISTS group_things (
						thing_id    UUID UNIQUE NOT NULL,
						group_id    UUID NOT NULL,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id) REFERENCES things (id) ON DELETE CASCADE,
						PRIMARY KEY (thing_id, group_id)
          )`,
					`CREATE TABLE IF NOT EXISTS group_channels (
						channel_id  UUID UNIQUE NOT NULL,
						group_id    UUID NOT NULL,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
						FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE,
						PRIMARY KEY (channel_id, group_id)
          )`,
				},
				Down: []string{
					"DROP TABLE connections",
					"DROP TABLE things",
					"DROP TABLE channels",
					"DROP TABLE groups",
					"DROP TABLE group_channels",
					"DROP TABLE group_things",
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
