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
				Id: "clients_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS clients (
						id            UUID PRIMARY KEY,
						group_id      UUID NOT NULL,
						thing_id      UUID NOT NULL,
						name          VARCHAR(254) NOT NULL,
						ip_address    VARCHAR(64) NOT NULL,
						port          VARCHAR(16) NOT NULL,
						slave_id      SMALLINT NOT NULL,
						function_code VARCHAR(32) NOT NULL,
						start_address INTEGER NOT NULL,
						read_length   INTEGER NOT NULL,
						scheduler     JSONB NOT NULL,
						data_fields   JSONB,
						metadata      JSONB,
						CONSTRAINT    unique_thing_name UNIQUE (thing_id, name)
					)`,
				},
				Down: []string{"DROP TABLE clients"},
			},
			{
				Id: "clients_2",
				Up: []string{
					`ALTER TABLE clients DROP COLUMN IF EXISTS start_address`,
					`ALTER TABLE clients DROP COLUMN IF EXISTS read_length`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
