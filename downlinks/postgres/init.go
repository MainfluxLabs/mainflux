package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
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
				Id: "downlinks_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS downlinks (
						id          UUID PRIMARY KEY,
						group_id    UUID NOT NULL,
						thing_id    UUID NOT NULL,
						name        VARCHAR(254) NOT NULL,
						url         VARCHAR(254) NOT NULL,
						method      VARCHAR(10) NOT NULL,
						payload     BYTEA,  
						headers     JSONB,
						scheduler   VARCHAR(254) NOT NULL,     
						time_zone   VARCHAR(254) NOT NULL,
						cron_id     VARCHAR(128) NOT NULL,
						metadata    JSONB,
						CONSTRAINT  unique_group_name UNIQUE (thing_id, name)
					)`,
				},
				Down: []string{"DROP TABLE downlinks"},
			},
			{
				Id: "downlinks_2",
				Up: []string{
					`ALTER TABLE downlinks ALTER COLUMN url TYPE VARCHAR(1024)`,
				},
			},
			{
				Id: "downlinks_3",
				Up: []string{
					`ALTER TABLE downlinks DROP COLUMN scheduler`,
					`ALTER TABLE downlinks DROP COLUMN time_zone`,
					`ALTER TABLE downlinks ADD COLUMN start_time VARCHAR(64)`,
					`ALTER TABLE downlinks ADD COLUMN end_time VARCHAR(64)`,
					`ALTER TABLE downlinks ADD COLUMN time_format VARCHAR(64)`,
					`ALTER TABLE downlinks ADD COLUMN scheduler JSONB`,
				},
			},
			{
				Id: "downlinks_4",
				Up: []string{
					`ALTER TABLE downlinks DROP COLUMN cron_id`,
					`UPDATE downlinks SET scheduler = jsonb_set(scheduler, '{time_zone}', '"UTC"')
					 WHERE scheduler->>'time_zone' = '' OR NOT scheduler ? 'time_zone'`,
				},
			},
			{
				Id: "downlinks_5",
				Up: []string{
					`ALTER TABLE downlinks ADD COLUMN IF NOT EXISTS forecast BOOLEAN DEFAULT FALSE`,
				},
			},
			{
				Id: "downlinks_6",
				Up: []string{
					`ALTER TABLE downlinks ADD COLUMN IF NOT EXISTS time_filter JSONB`,
					`UPDATE downlinks
					 SET time_filter = jsonb_build_object(
						'start_param', start_time,
						'end_param', end_time,
						'format', time_format,
						'forecast', COALESCE(forecast, FALSE)
					)
					WHERE time_filter IS NULL
					AND (start_time IS NOT NULL OR end_time IS NOT NULL)`,
				},
			},
			{
				Id: "downlinks_7",
				Up: []string{
					`ALTER TABLE downlinks DROP COLUMN IF EXISTS start_time`,
					`ALTER TABLE downlinks DROP COLUMN IF EXISTS end_time`,
					`ALTER TABLE downlinks DROP COLUMN IF EXISTS time_format`,
					`ALTER TABLE downlinks DROP COLUMN IF EXISTS forecast`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
