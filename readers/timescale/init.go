// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

// Config defines the options that are used when connecting to a TimescaleSQL instance
type Config struct {
	Host          string
	Port          string
	User          string
	Pass          string
	Name          string
	SSLMode       string
	SSLCert       string
	SSLKey        string
	SSLRootCert   string
	ChunkInterval string
}

// Connect creates a connection to the TimescaleSQL instance and applies any
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

	if err := setChunkInterval(db, cfg.ChunkInterval); err != nil {
		return nil, err
	}

	return db, nil
}

func setChunkInterval(db *sqlx.DB, interval string) error {
	if interval == "" {
		return nil
	}

	d, err := time.ParseDuration(interval)
	if err != nil {
		return fmt.Errorf("invalid chunk interval %q: %w", interval, err)
	}
	if d <= 0 {
		return fmt.Errorf("invalid chunk interval %q: must be positive", interval)
	}

	q := `SELECT set_chunk_time_interval(CAST($1 AS regclass), CAST($2 AS bigint))`
	for _, table := range []string{"senml", "json"} {
		if _, err := db.Exec(q, table, d.Nanoseconds()); err != nil {
			return err
		}
	}

	return nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "messages_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS senml (
						time          BIGINT NOT NULL,
						subtopic      VARCHAR(254),
						publisher     UUID,
						protocol      TEXT,
						name          VARCHAR(254),
						unit          TEXT,
						value         FLOAT,
						string_value  TEXT,
						bool_value    BOOL,
						data_value    BYTEA,
						sum           FLOAT,
						update_time   FLOAT,
						PRIMARY KEY   (time, publisher, subtopic, name)
					);
					SELECT create_hypertable('senml', 'time', create_default_indexes => FALSE, chunk_time_interval => 604800000000000, if_not_exists => TRUE);`,
					`CREATE TABLE IF NOT EXISTS json (
						created       BIGINT NOT NULL,
						subtopic      VARCHAR(254),
						publisher     VARCHAR(254),
						protocol      TEXT,
						payload       JSONB,
						PRIMARY KEY   (created, publisher, subtopic)
					);
					SELECT create_hypertable('json', 'created', create_default_indexes => FALSE, chunk_time_interval => 604800000000000, if_not_exists => TRUE);`,
					`CREATE INDEX IF NOT EXISTS idx_json_created ON json(created DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_json_publisher_created ON json(publisher, created DESC)`,
					`CREATE INDEX IF NOT EXISTS idx_senml_publisher_time ON senml(publisher, time DESC)`,
				},
				Down: []string{
					"DROP TABLE senml",
					"DROP TABLE json",
				},
			},
			{
				Id: "messages_2",
				Up: []string{
					`ALTER TABLE json DROP CONSTRAINT IF EXISTS json_pkey`,
					`ALTER TABLE json ADD COLUMN IF NOT EXISTS payload_hash INTEGER`,
					`CREATE UNIQUE INDEX IF NOT EXISTS idx_json_dedup ON json(created, publisher, subtopic, payload_hash)`,
				},
				Down: []string{
					"DROP INDEX IF EXISTS idx_json_dedup",
					"ALTER TABLE json DROP COLUMN IF EXISTS payload_hash",
				},
			},
			{
				Id: "messages_3",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_senml_publisher_name_time ON senml(publisher, name, time DESC)`,
					`DROP INDEX IF EXISTS idx_json_created`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_senml_publisher_name_time`,
					`CREATE INDEX IF NOT EXISTS idx_json_created ON json(created DESC)`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
