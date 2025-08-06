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
// unapplied database migrations. A non-nil error is returned to indicate failure.
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
				Id: "auth_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS users_roles (
						role    VARCHAR(12) CHECK (role IN ('root', 'admin')),
						user_id UUID NOT NULL,
						PRIMARY KEY (user_id)
					)`,
					`CREATE TABLE IF NOT EXISTS keys (
						id          VARCHAR(254) NOT NULL,
						type        SMALLINT,
						subject     VARCHAR(254) NOT NULL,
						issuer_id   UUID NOT NULL,
						issued_at   TIMESTAMP NOT NULL,
						expires_at  TIMESTAMP,
						PRIMARY KEY (id, issuer_id)
					)`,
					`CREATE TABLE IF NOT EXISTS orgs (
						id          UUID UNIQUE NOT NULL,
						owner_id    UUID,
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						PRIMARY KEY (id, owner_id)
					)`,
					`CREATE TABLE IF NOT EXISTS member_relations (
						member_id   UUID NOT NULL,
						org_id      UUID NOT NULL,
						role        VARCHAR(10) NOT NULL,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE,
						PRIMARY KEY (member_id, org_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS users_roles`,
					`DROP TABLE IF EXISTS keys`,
					`DROP TABLE IF EXISTS orgs`,
					`DROP TABLE IF EXISTS member_relations`,
				},
			},
			{
				Id: "auth_2",
				Up: []string{
					`ALTER TABLE member_relations RENAME TO org_memberships`,
					`ALTER TABLE org_memberships RENAME CONSTRAINT member_relations_org_id_fkey TO org_memberships_org_id_fkey`,
					`ALTER TABLE org_memberships RENAME CONSTRAINT member_relations_pkey TO org_memberships_pkey`,
				},
				Down: []string{},
			},
			{
				Id: "auth_3",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS invites (
						id UUID NOT NULL,
						invitee_id UUID,
						invitee_email VARCHAR,
						inviter_id UUID NOT NULL,
						org_id UUID NOT NULL,
						invitee_role VARCHAR(12) NOT NULL,
						created_at TIMESTAMPTZ,
						expires_at TIMESTAMPTZ,
						FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE CASCADE,
						PRIMARY KEY (id),
						UNIQUE (invitee_id, org_id),
						UNIQUE (invitee_email, org_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS invites`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
