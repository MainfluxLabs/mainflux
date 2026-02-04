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
				Id: "mqtt_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS subscriptions (
					    subtopic    VARCHAR(1024),
					    group_id  	UUID,
					    thing_id    UUID,
					    client_id   VARCHAR(256) UNIQUE,
					    status      VARCHAR(128),
					    created_at  FLOAT,
					    PRIMARY KEY (client_id, subtopic, group_id, thing_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS subscriptions`,
				},
			},
			{
				Id: "mqtt_2",
				Up: []string{
					`ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_client_id_key`,
				},
			},
			{
				Id: "mqtt_3",
				Up: []string{
					`ALTER TABLE subscriptions DROP COLUMN IF EXISTS status`,
					`DELETE FROM subscriptions s1
					 USING subscriptions s2
					 WHERE s1.subtopic = s2.subtopic
						AND s1.group_id = s2.group_id
						AND s1.thing_id = s2.thing_id
						AND s1.created_at > s2.created_at`,
					`ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_pkey`,
					`ALTER TABLE subscriptions ADD PRIMARY KEY (subtopic, group_id, thing_id)`,
					`ALTER TABLE subscriptions DROP COLUMN IF EXISTS client_id`,
				},
			},
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
