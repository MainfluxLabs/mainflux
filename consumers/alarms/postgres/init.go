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
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

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
				Id: "alarms_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS alarms (
						id          UUID PRIMARY KEY,
						thing_id    UUID NOT NULL,
						group_id    UUID NOT NULL,
						rule_id     UUID NOT NULL,
						subtopic    VARCHAR(254),
						protocol    TEXT,
						payload     JSONB,
						created     BIGINT
					)`,
				},
				Down: []string{"DROP TABLE alarms"},
			},
			{
				Id: "alarms_2",
				Up: []string{
					`ALTER TABLE alarms ALTER COLUMN rule_id DROP NOT NULL;`,
					`ALTER TABLE alarms ADD COLUMN script_id UUID;`,
					`ALTER TABLE alarms ADD CONSTRAINT alarm_single_origin CHECK (num_nonnulls(rule_id, script_id) = 1)`,
				},
				Down: []string{
					`ALTER TABLE alarms ALTER COLUMN rule_id SET NOT NULL;`,
					`ALTER TABLE alarms DROP COLUMN IF EXISTS script_id;`,
					`ALTER TABLE alarms DROP CONSTRAINT IF EXISTS alarm_single_origin;`,
				},
			},
			{
				Id: "alarms_3",
				Up: []string{
					`ALTER TABLE alarms ADD COLUMN level  SMALLINT NOT NULL DEFAULT 1;`,
					`ALTER TABLE alarms ADD COLUMN status VARCHAR(10) NOT NULL DEFAULT 'active';`,
					`ALTER TABLE alarms ADD COLUMN rule   JSONB;`,
					`ALTER TABLE alarms DROP COLUMN IF EXISTS payload;`,
				},
				Down: []string{
					`ALTER TABLE alarms DROP COLUMN IF EXISTS level;`,
					`ALTER TABLE alarms DROP COLUMN IF EXISTS status;`,
					`ALTER TABLE alarms DROP COLUMN IF EXISTS rule;`,
					`ALTER TABLE alarms ADD COLUMN payload JSONB;`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
