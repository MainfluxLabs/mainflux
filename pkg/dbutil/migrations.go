// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package dbutil

import (
	"database/sql"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
)

// PadMigrationIDs renames applied migration IDs to their zero-padded form.
// sql-migrate orders non-numeric IDs lexically, so unpadded IDs apply out of order.
func PadMigrationIDs(db *sql.DB, migrations []*migrate.Migration) error {
	var exists bool
	row := db.QueryRow(`SELECT to_regclass('gorp_migrations') IS NOT NULL`)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return nil
	}

	for i, m := range migrations {
		// IDs from the tenth migration onward already have two digits.
		if i >= 9 {
			break
		}

		oldID := strings.Replace(m.Id, "_0", "_", 1)
		query := `UPDATE gorp_migrations SET id = $1 WHERE id = $2`
		if _, err := db.Exec(query, m.Id, oldID); err != nil {
			return err
		}
	}

	return nil
}
