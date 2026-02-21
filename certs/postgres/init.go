// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

const primaryKey = "primary_key"

// ErrMigrate indicates error during database migrations.
var ErrMigrate = errors.New("error executing database migrations")

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
		mErr, ok := err.(*migrate.TxError)
		if ok && mErr.Migration.Id == primaryKey {
			return db, ErrMigrate
		}
		return nil, err
	}

	return db, nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS certs (
						thing_id         UUID NOT NULL,
						serial           VARCHAR(64) NOT NULL UNIQUE,
						expires_at       TIMESTAMPTZ NOT NULL,
						client_cert      TEXT NOT NULL,
						client_key       TEXT NOT NULL,
						issuing_ca       TEXT NOT NULL,
						ca_chain         TEXT[],
						private_key_type TEXT NOT NULL,
						created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						PRIMARY KEY      (thing_id, serial)
					);`,
					`CREATE INDEX idx_certs_serial ON certs(serial);`,
					`CREATE INDEX idx_certs_expires_at ON certs(expires_at);`,
					`CREATE TABLE IF NOT EXISTS revoked_certs (
						thing_id         UUID NOT NULL,
						serial           VARCHAR(64) NOT NULL UNIQUE,
						revoked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
					);`,
					`CREATE INDEX idx_revoked_certs_thing_id ON revoked_certs(thing_id);`,
					`CREATE INDEX idx_revoked_certs_revoked_at ON revoked_certs(revoked_at);`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS revoked_certs;",
					"DROP TABLE IF EXISTS certs;",
				},
			},
			{
				Id: "certs_2",
				Up: []string{
					`ALTER TABLE certs ADD COLUMN IF NOT EXISTS key_bits INTEGER NOT NULL DEFAULT 0;`,
				},
				Down: []string{
					`ALTER TABLE certs DROP COLUMN IF EXISTS key_bits;`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
