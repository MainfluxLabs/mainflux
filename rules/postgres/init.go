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
		},
	}
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
