// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
)

// Config defines the options that are used when connecting to a TimescaleSQL instance
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

// Connect creates a connection to the TimescaleSQL instance. Unlike the
// writer, it does not run migrations — the reader has no write access and
// does not own the schema. A non-nil error is returned to indicate failure.
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	return db, nil
}
