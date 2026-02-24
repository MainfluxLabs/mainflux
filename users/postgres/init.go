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
				Id: "users_1",
				Up: []string{
					`CREATE TYPE user_status AS ENUM ('enabled', 'disabled')`,
					`CREATE TABLE IF NOT EXISTS users (
						id          UUID UNIQUE NOT NULL,
						email       VARCHAR(254) UNIQUE NOT NULL,
						password    CHAR(60) NOT NULL,
						metadata    JSONB,
						status      USER_STATUS NOT NULL DEFAULT 'enabled',
						PRIMARY KEY (id)
					)`,
				},
				Down: []string{
					"DROP TABLE users",
				},
			},
			{
				Id: "users_2",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS verifications (
						email      VARCHAR(254) NOT NULL,
						password   CHAR(60) NOT NULL,
						token      UUID UNIQUE NOT NULL,
						created_at TIMESTAMPTZ NOT NULL,
						expires_at TIMESTAMPTZ NOT NULL
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS verifications`,
				},
			},
			{
				Id: "users_3",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS platform_invites (
						id            UUID NOT NULL,
						invitee_email VARCHAR NOT NULL,
						created_at    TIMESTAMPTZ,
						expires_at    TIMESTAMPTZ,
						state         VARCHAR DEFAULT 'pending' NOT NULL
 					)`,
					`CREATE UNIQUE INDEX unique_invitee_email_pending on platform_invites (invitee_email) WHERE state='pending'`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS platform_invites`,
					`DROP INDEX IF EXISTS unique_invitee_email_pending`,
				},
			},
			{
				Id: "users_4",
				Up: []string{
					`ALTER TABLE users ALTER COLUMN password DROP NOT NULL`,

					`CREATE TABLE user_identities (
						user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
						provider VARCHAR(32) NOT NULL,
						provider_user_id VARCHAR(128) NOT NULL,
						PRIMARY KEY (user_id, provider),
						UNIQUE (provider, provider_user_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS user_identities`,
					`UPDATE users SET password = '' WHERE password IS NULL`,
					`ALTER TABLE users ALTER COLUMN password SET NOT NULL`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
