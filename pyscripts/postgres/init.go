// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"

	migrate "github.com/rubenv/sql-migrate"
)

// Config defines the options that are used when connecting to a PostgreSQL instance.
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
// unapplied database migrations. A non-nil error is returned to indicate failure.
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert,
	)

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
				Id: "pyscripts_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS scripts (
						id          UUID NOT NULL,
						group_id    UUID NOT NULL,
						name        VARCHAR NOT NULL,
						description TEXT NOT NULL DEFAULT '',
						source      TEXT NOT NULL,
						sha256      CHAR(64) NOT NULL,
						created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						PRIMARY KEY (id)
					);`,
					`CREATE TABLE IF NOT EXISTS script_runs (
						id            UUID NOT NULL,
						script_id     UUID NOT NULL,
						script_sha256 CHAR(64) NOT NULL,
						return_value  JSONB NULL,
						logs          JSONB NOT NULL,
						started_at    TIMESTAMPTZ NOT NULL,
						finished_at   TIMESTAMPTZ NOT NULL,
						status        TEXT NOT NULL,
						error         TEXT NULL,
						PRIMARY KEY (id),
						FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE CASCADE
					);`,
					`CREATE INDEX IF NOT EXISTS idx_scripts_group_id ON scripts (group_id);`,
					`CREATE INDEX IF NOT EXISTS idx_script_runs_script_id ON script_runs (script_id);`,
					`CREATE INDEX IF NOT EXISTS idx_script_runs_started_at ON script_runs (started_at);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS script_runs;`,
					`DROP TABLE IF EXISTS scripts;`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
