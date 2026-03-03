// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ certs.Repository = (*certsRepository)(nil)

type certsRepository struct {
	db dbutil.Database
}

func NewRepository(db dbutil.Database) certs.Repository {
	return &certsRepository{db: db}
}

func (cr certsRepository) RetrieveAll(ctx context.Context, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, serial, expires_at, client_cert, client_key, issuing_ca,
	      ca_chain, private_key_type, key_bits FROM certs ORDER BY expires_at LIMIT :limit OFFSET :offset;`

	params := map[string]any{
		"limit":  limit,
		"offset": offset,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return certs.Page{}, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		var dbcrt dbCert
		if err := rows.StructScan(&dbcrt); err != nil {
			return certs.Page{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		certificates = append(certificates, toCert(dbcrt))
	}

	q = `SELECT COUNT(*) FROM certs`
	total, err := dbutil.Total(ctx, cr.db, q, params)
	if err != nil {
		return certs.Page{}, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}

	return certs.Page{
		Total: total,
		Certs: certificates,
	}, nil
}

func (cr certsRepository) Save(ctx context.Context, cert certs.Cert) (string, error) {
	q := `INSERT INTO certs (thing_id, serial, expires_at, client_cert, client_key,
	      issuing_ca, ca_chain, private_key_type, key_bits)
	      VALUES (:thing_id, :serial, :expires_at, :client_cert, :client_key,
	      :issuing_ca, :ca_chain, :private_key_type, :key_bits)`

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	dbcrt := toDBCert(cert)

	if _, err := tx.NamedExecContext(ctx, q, dbcrt); err != nil {
		return "", cr.handlePgError(err, dbutil.ErrCreateEntity)
	}

	if err := tx.Commit(); err != nil {
		return "", errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return cert.Serial, nil
}

func (cr certsRepository) Remove(ctx context.Context, serial string) error {
	cert, err := cr.RetrieveBySerial(ctx, serial)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO revoked_certs (serial, thing_id, revoked_at) 
	            VALUES (:serial, :thing_id, NOW())`
	revokeParams := map[string]any{
		"serial":   serial,
		"thing_id": cert.ThingID,
	}
	if _, err := tx.NamedExecContext(ctx, q, revokeParams); err != nil {
		return cr.handlePgError(err, dbutil.ErrRemoveEntity)
	}

	q = `DELETE FROM certs WHERE serial = :serial`
	deleteParams := map[string]any{
		"serial": serial,
	}
	if _, err := tx.NamedExecContext(ctx, q, deleteParams); err != nil {
		return cr.handlePgError(err, dbutil.ErrRemoveEntity)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (cr certsRepository) RetrieveRevokedCerts(ctx context.Context) ([]certs.RevokedCert, error) {
	q := `SELECT serial, revoked_at, thing_id FROM revoked_certs ORDER BY revoked_at DESC`

	rows, err := cr.db.NamedQueryContext(ctx, q, map[string]any{})
	if err != nil {
		return nil, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}
	defer rows.Close()

	var revokedCerts []certs.RevokedCert
	for rows.Next() {
		var cert certs.RevokedCert
		if err := rows.StructScan(&cert); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		revokedCerts = append(revokedCerts, cert)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return revokedCerts, nil
}

func (cr certsRepository) RetrieveByThing(ctx context.Context, thingID string, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, serial, expires_at, client_cert, client_key, issuing_ca,
	      ca_chain, private_key_type, key_bits FROM certs
	      WHERE thing_id = :thing_id ORDER BY expires_at LIMIT :limit OFFSET :offset;`

	params := map[string]any{
		"thing_id": thingID,
		"limit":    limit,
		"offset":   offset,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return certs.Page{}, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		var dbcrt dbCert
		if err := rows.StructScan(&dbcrt); err != nil {
			return certs.Page{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		certificates = append(certificates, toCert(dbcrt))
	}

	q = `SELECT COUNT(*) FROM certs WHERE thing_id = :thing_id`
	total, err := dbutil.Total(ctx, cr.db, q, params)
	if err != nil {
		return certs.Page{}, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}

	return certs.Page{
		Total: total,
		Certs: certificates,
	}, nil
}

func (cr certsRepository) RetrieveBySerial(ctx context.Context, serial string) (certs.Cert, error) {
	q := `SELECT thing_id, serial, expires_at, client_cert, client_key, issuing_ca,
	      ca_chain, private_key_type, key_bits FROM certs WHERE serial = :serial`

	params := map[string]any{
		"serial": serial,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return certs.Cert{}, cr.handlePgError(err, dbutil.ErrRetrieveEntity)
	}
	defer rows.Close()

	if !rows.Next() {
		return certs.Cert{}, errors.Wrap(dbutil.ErrNotFound, sql.ErrNoRows)
	}

	var dbcrt dbCert
	if err := rows.StructScan(&dbcrt); err != nil {
		return certs.Cert{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toCert(dbcrt), nil
}

func (cr certsRepository) handlePgError(err error, wrapErr error) error {
	pgErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return errors.Wrap(wrapErr, errors.New("error conflict"))
		case pgerrcode.InvalidTextRepresentation:
			return errors.Wrap(wrapErr, err)
		case pgerrcode.UndefinedTable:
			return errors.Wrap(wrapErr, err)
		default:
			return errors.Wrap(wrapErr, err)
		}
	}
	return errors.Wrap(wrapErr, err)
}

type dbCert struct {
	ThingID        string      `db:"thing_id"`
	Serial         string      `db:"serial"`
	ExpiresAt      time.Time   `db:"expires_at"`
	ClientCert     string      `db:"client_cert"`
	ClientKey      string      `db:"client_key"`
	IssuingCA      string      `db:"issuing_ca"`
	CAChain        StringArray `db:"ca_chain"`
	PrivateKeyType string      `db:"private_key_type"`
	KeyBits        int         `db:"key_bits"`
}

func toDBCert(c certs.Cert) dbCert {
	return dbCert{
		ThingID:        c.ThingID,
		Serial:         c.Serial,
		ExpiresAt:      c.ExpiresAt,
		ClientCert:     c.ClientCert,
		ClientKey:      c.ClientKey,
		IssuingCA:      c.IssuingCA,
		CAChain:        StringArray(c.CAChain),
		PrivateKeyType: c.PrivateKeyType,
		KeyBits:        c.KeyBits,
	}
}

func toCert(cdb dbCert) certs.Cert {
	return certs.Cert{
		ThingID:        cdb.ThingID,
		Serial:         cdb.Serial,
		ExpiresAt:      cdb.ExpiresAt,
		ClientCert:     cdb.ClientCert,
		ClientKey:      cdb.ClientKey,
		IssuingCA:      cdb.IssuingCA,
		CAChain:        []string(cdb.CAChain),
		PrivateKeyType: cdb.PrivateKeyType,
		KeyBits:        cdb.KeyBits,
	}
}

type StringArray []string

func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = []string{}
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return a.scanBytes(v)
	case string:
		return a.scanBytes([]byte(v))
	case []string:
		*a = v
		return nil
	default:
		return fmt.Errorf("cannot scan %T to StringArray", src)
	}
}

func (a *StringArray) scanBytes(src []byte) error {
	str := string(src)

	if str == "{}" || str == "" {
		*a = []string{}
		return nil
	}

	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")

	if str == "" {
		*a = []string{}
		return nil
	}

	parts := strings.Split(str, ",")
	result := make([]string, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"")
		result[i] = part
	}

	*a = result
	return nil
}

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}

	quoted := make([]string, len(a))
	for i, s := range a {
		s = strings.ReplaceAll(s, `"`, `\"`)
		quoted[i] = `"` + s + `"`
	}

	return "{" + strings.Join(quoted, ",") + "}", nil
}
