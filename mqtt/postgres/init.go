package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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

	db, err := sqlx.Open("postgres", url)
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
				Id: "mqtt_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS subscriptions (
					    id          VARCHAR(254) PRIMARY KEY,
                                  owner_id    VARCHAR(254) NOT NULL,
					    subtopic    VARCHAR(1024),
					    channel_id  UUID,
					    thing_id    UUID,
					    time  	    FLOAT,
					    PRIMARY KEY (subtopic, channel_id, thing_id)
					)`,
				},
				Down: []string{
				    `DROP TABLE IF EXISTS subscriptions`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
