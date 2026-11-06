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
	// The schema is defined in a single migration. Databases created by earlier
	// releases already have rules_1 recorded and are left untouched, because
	// sql-migrate skips a migration whose identifier is already applied.
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "rules_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rules (
						id           UUID PRIMARY KEY,
						group_id     UUID NOT NULL,
						name         VARCHAR(254) NOT NULL,
						description  VARCHAR(1024),
						conditions   JSONB NOT NULL,
						actions      JSONB NOT NULL,
						operator     VARCHAR(3) NOT NULL DEFAULT '',
						input_type   TEXT NOT NULL DEFAULT 'message',
						input_config JSONB NOT NULL DEFAULT '{}'
					)`,
					`CREATE TABLE IF NOT EXISTS rules_things (
						rule_id  UUID NOT NULL REFERENCES rules (id) ON DELETE CASCADE,
						thing_id UUID NOT NULL,
						PRIMARY KEY (rule_id, thing_id)
					)`,
					`CREATE TABLE IF NOT EXISTS lua_scripts (
						id          UUID PRIMARY KEY,
						group_id    UUID NOT NULL,
						script      VARCHAR(65535) NOT NULL,
						name        VARCHAR NOT NULL,
						description VARCHAR NOT NULL
					)`,
					`CREATE TABLE IF NOT EXISTS lua_script_runs (
						id          UUID PRIMARY KEY,
						script_id   UUID NOT NULL REFERENCES lua_scripts (id) ON DELETE CASCADE,
						thing_id    UUID NOT NULL,
						logs        JSONB NOT NULL,
						started_at  TIMESTAMPTZ NOT NULL,
						finished_at TIMESTAMPTZ NOT NULL,
						status      TEXT NOT NULL,
						error       TEXT NULL,
						rule_id     UUID NULL REFERENCES rules (id) ON DELETE CASCADE
					)`,
					`CREATE INDEX IF NOT EXISTS idx_lua_script_runs_thing_id ON lua_script_runs (thing_id)`,
					`CREATE INDEX IF NOT EXISTS idx_lua_script_runs_script_id ON lua_script_runs (script_id)`,
					`CREATE INDEX IF NOT EXISTS idx_lua_script_runs_rule_id ON lua_script_runs (rule_id)`,
					`CREATE INDEX IF NOT EXISTS idx_rules_group_input_type ON rules (group_id, input_type)`,
					`CREATE INDEX IF NOT EXISTS idx_rules_things_thing_id ON rules_things (thing_id)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS lua_script_runs`,
					`DROP TABLE IF EXISTS lua_scripts`,
					`DROP TABLE IF EXISTS rules_things`,
					`DROP TABLE IF EXISTS rules`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
