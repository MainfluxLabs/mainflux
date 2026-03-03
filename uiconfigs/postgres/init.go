package postgres

import (
	"fmt"
	"log"

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

	// Temporary migration necessary due to subsequent addition of the group_id field to the thing_configs table.
	if err := populateGroupID(db, cfg); err != nil {
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

// Populate missing group_id values in thing_configs by mapping it from the things table.
func populateGroupID(fsDB *sqlx.DB, cfg Config) error {
	var ids []string
	if err := fsDB.Select(&ids, `SELECT DISTINCT thing_id FROM thing_configs WHERE group_id IS NULL`); err != nil {
		return err
	}
	if len(ids) == 0 {
		log.Printf("no rows without group_id in thing_configs")
		return nil
	}

	dsn := fmt.Sprintf(
		"host=things-db port=%s user=%s dbname=things password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		cfg.Port, cfg.User, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert,
	)

	thDB, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer thDB.Close()

	tx, err := fsDB.Beginx()
	if err != nil {
		return err
	}

	batchSize := 500
	var total int64

	type thing struct {
		ID      string `db:"id"`
		GroupID string `db:"group_id"`
	}

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batchIDs := ids[i:end]

		var ths []thing
		query, args, err := sqlx.In(`SELECT id, group_id FROM things WHERE id IN (?)`, batchIDs)
		if err != nil {
			return err
		}

		query = thDB.Rebind(query)
		if err := thDB.Select(&ths, query, args...); err != nil {
			log.Printf("failed SELECT on things table for batch %d-%d: %v", i, end, err)
			tx.Rollback()
			return err
		}

		if len(ths) == 0 {
			continue
		}

		values := ""
		params := make([]interface{}, 0, len(ths)*2)

		for idx, th := range ths {
			if idx > 0 {
				values += ", "
			}

			values += fmt.Sprintf("($%d::uuid, $%d::uuid)", idx*2+1, idx*2+2)
			params = append(params, th.ID, th.GroupID)
		}

		updateQuery := fmt.Sprintf(`
					UPDATE thing_configs AS tf
					SET group_id = data.group_id
					FROM (VALUES %s) AS data(thing_id, group_id)
					WHERE tf.thing_id = data.thing_id
					`, values)

		result, err := tx.Exec(updateQuery, params...)
		if err != nil {
			tx.Rollback()
			return err
		}

		affected, _ := result.RowsAffected()
		total += affected

		log.Printf("processed batch %d-%d: %d rows updated", i, end, affected)
	}

	// Delete orphaned thing_configs where the thing no longer exists in the things table
	result, err := tx.Exec(`DELETE FROM thing_configs WHERE group_id IS NULL`)
	if err != nil {
		tx.Rollback()
		return err
	}
	deleted, _ := result.RowsAffected()
	if deleted > 0 {
		log.Printf("deleted %d orphaned thing_configs (thing no longer exists)", deleted)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("completed group_id backfill in thing_configs from things table for %d rows", total)
	return nil
}
