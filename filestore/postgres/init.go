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
				Id: "filestore_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS things_files (
							file_name   VARCHAR(1024) NOT NULL,
							file_class  VARCHAR(254) NOT NULL,
							file_format VARCHAR(254) NOT NULL,
							thing_id    UUID NOT NULL,
							time  	    FLOAT NOT NULL,
							metadata    JSONB,
							PRIMARY KEY (thing_id, file_name, file_class, file_format)
					)`,
					`CREATE TABLE IF NOT EXISTS groups_files (
							file_name   VARCHAR(1024) NOT NULL,
							file_class  VARCHAR(254) NOT NULL,
							file_format VARCHAR(254) NOT NULL,
							group_id    UUID NOT NULL,
							time  	    FLOAT NOT NULL,
							metadata    JSONB,
							PRIMARY KEY (group_id, file_name, file_class, file_format)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS things_files`,
					`DROP TABLE IF EXISTS groups_files`,
				},
			},
			{
				Id: "filestore_2",
				Up: []string{
					`ALTER TABLE things_files ADD COLUMN group_id UUID;`,
				},
				Down: []string{
					`ALTER TABLE things_files DROP COLUMN group_id`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
