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
				Id: "rules_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rules (
						id          UUID PRIMARY KEY,
						profile_id  UUID NOT NULL,
						group_id    UUID NOT NULL, 
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						condition   JSONB NOT NULL,
						actions     JSONB NOT NULL
					)`,
				},
				Down: []string{"DROP TABLE rules"},
			},
			{
				Id: "rules_2",
				Up: []string{
					`ALTER TABLE rules RENAME COLUMN condition TO conditions`,
					`ALTER TABLE rules ADD COLUMN operator VARCHAR(3) NOT NULL DEFAULT ''`,
					`UPDATE rules
					 SET conditions = jsonb_build_array(jsonb_set(conditions, '{comparator}', conditions->'operator') - 'operator')
					 WHERE jsonb_typeof(conditions) = 'object'`,
				},
			},
			{
				Id: "rules_3",
				Up: []string{
					`ALTER TABLE rules DROP COLUMN IF EXISTS profile_id;`,
				},
			},
			{
				Id: "rules_4",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rules_things (
						rule_id   UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
						thing_id  UUID NOT NULL,
						PRIMARY KEY (rule_id, thing_id)
					);`,
				},
			},
			{
				Id: "rules_5",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS lua_scripts (
						id          UUID NOT NULL,
						group_id    UUID NOT NULL,
						script      VARCHAR(65535) NOT NULL,
						name        VARCHAR NOT NULL,
						description VARCHAR NOT NULL,
						PRIMARY KEY (id)
					);`,
					`CREATE TABLE IF NOT EXISTS lua_scripts_things (
						thing_id      UUID NOT NULL,
						lua_script_id UUID NOT NULL,
						PRIMARY KEY (thing_id, lua_script_id),
						FOREIGN KEY (lua_script_id) REFERENCES lua_scripts (id) ON DELETE CASCADE
					);`,
					`CREATE TABLE IF NOT EXISTS lua_script_runs (
						id          UUID NOT NULL,
						script_id   UUID NOT NULL,
						thing_id    UUID NOT NULL,
						logs        JSONB NOT NULL,
						started_at  TIMESTAMPTZ NOT NULL,
						finished_at TIMESTAMPTZ NOT NULL,
						status      TEXT NOT NULL,
						error       TEXT NULL,
						PRIMARY KEY (id),
						FOREIGN KEY (script_id) REFERENCES lua_scripts (id) ON DELETE CASCADE
					);`,
					`CREATE INDEX IF NOT EXISTS idx_lua_script_runs_thing_id ON lua_script_runs(thing_id)`,
					`CREATE INDEX IF NOT EXISTS idx_lua_script_runs_script_id ON lua_script_runs(script_id)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS lua_scripts;`,
					`DROP TABLE IF EXISTS lua_scripts_things;`,
					`DROP TABLE IF EXISTS lua_script_runs;`,
					`DROP INDEX IF EXISTS idx_lua_script_runs_thing_id;`,
					`DROP INDEX IF EXISTS idx_lua_script_runs_script_id;`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
