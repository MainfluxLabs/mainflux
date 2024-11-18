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
					`CREATE TABLE IF NOT EXISTS groups (
						id          UUID UNIQUE NOT NULL,
						owner_id    UUID NOT NULL,
						org_id      UUID NOT NULL,
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						PRIMARY KEY (id, owner_id)
					)`,
					`CREATE TABLE IF NOT EXISTS group_policies (
						group_id    UUID NOT NULL,
						member_id   UUID NOT NULL,
						policy      VARCHAR(15),
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
						PRIMARY KEY (group_id, member_id)
					)`,
					`CREATE TABLE IF NOT EXISTS things (
						id          UUID UNIQUE NOT NULL,
						owner_id    UUID NOT NULL,
						group_id    UUID NOT NULL,
						key         VARCHAR(4096) UNIQUE NOT NULL,
						name        VARCHAR(1024),
						metadata    JSONB,
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (id, owner_id)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						id          UUID UNIQUE NOT NULL,
						owner_id    UUID NOT NULL,
						group_id    UUID NOT NULL,
						name        VARCHAR(1024),
						profile     JSONB,
						metadata    JSONB,
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (id, owner_id)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id  UUID NOT NULL,
						thing_id    UUID UNIQUE NOT NULL,
						FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id) REFERENCES things (id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, thing_id)
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
			{
				Id: "things_2",
				Up: []string{
					`ALTER TABLE IF EXISTS group_policies RENAME TO group_roles;
						ALTER TABLE IF EXISTS group_roles RENAME COLUMN policy to role;
						UPDATE group_roles SET role = REPLACE(role, 'r', 'viewer');
						UPDATE group_roles SET role = REPLACE(role, 'r_w', 'admin');`,
				},
				Down: []string{
					"DROP TABLE group_roles",
				},
			},
			{
				Id: "things_3",
				Up: []string{
					`ALTER TABLE IF EXISTS channels RENAME TO profiles;
						ALTER TABLE IF EXISTS profiles RENAME COLUMN profile TO config;
						ALTER TABLE IF EXISTS connections RENAME COLUMN channel_id TO profile_id;
						ALTER INDEX IF EXISTS channels_pkey RENAME TO profiles_pkey;
						ALTER TABLE connections DROP CONSTRAINT IF EXISTS connections_channel_id_fkey;
						ALTER TABLE connections ADD CONSTRAINT connections_profile_id_fkey 
							FOREIGN KEY (profile_id) REFERENCES profiles (id) ON DELETE CASCADE ON UPDATE CASCADE;`,
				},
			},
			{
				Id: "things_4",
				Up: []string{
					`UPDATE things 
						SET name = CONCAT('th_', id)
						WHERE name IS NULL OR name = '';`,
					`UPDATE profiles 
						SET name = CONCAT('ch_', id) 
						WHERE name IS NULL OR name = '';`,
				},
			},
			{
				Id: "things_5",
				Up: []string{
					`ALTER TABLE groups DROP CONSTRAINT groups_pkey;
						ALTER TABLE groups DROP COLUMN IF EXISTS owner_id;
						ALTER TABLE groups ADD PRIMARY KEY (id);
						ALTER TABLE groups ADD CONSTRAINT org_name UNIQUE (org_id, name);`,
					`ALTER TABLE things DROP CONSTRAINT things_pkey;
						ALTER TABLE things DROP COLUMN IF EXISTS owner_id;
						ALTER TABLE things ADD PRIMARY KEY (id);
						ALTER TABLE things ADD CONSTRAINT group_name_ths UNIQUE (group_id, name);
						ALTER TABLE things ALTER COLUMN name SET NOT NULL;`,
					`ALTER TABLE profiles DROP CONSTRAINT profiles_pkey;
						ALTER TABLE profiles DROP COLUMN IF EXISTS owner_id;
						ALTER TABLE profiles ADD PRIMARY KEY (id);
						ALTER TABLE profiles ADD CONSTRAINT group_name_profs UNIQUE (group_id, name);
						ALTER TABLE profiles ALTER COLUMN name SET NOT NULL;`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
