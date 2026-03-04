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
				Id: "uiconfigs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS org_configs (
                        org_id    UUID PRIMARY KEY,
                        config    JSONB
					)`,
					`CREATE TABLE IF NOT EXISTS thing_configs (
						thing_id UUID PRIMARY KEY,
                        config   JSONB
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS org_configs`,
					`DROP TABLE IF EXISTS thing_configs`,
				},
			},
			{
				Id: "uiconfigs_2",
				Up: []string{
					`ALTER TABLE thing_configs ADD COLUMN group_id UUID;`,
				},
				Down: []string{
					`ALTER TABLE thing_configs DROP COLUMN group_id`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return err
	}

	return nil
}
