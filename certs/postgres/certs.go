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
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var _ certs.Repository = (*certsRepository)(nil)

type certsRepository struct {
	db  *sqlx.DB
	log logger.Logger
}

func NewRepository(db *sqlx.DB, log logger.Logger) certs.Repository {
	return &certsRepository{db: db, log: log}
}

func (cr certsRepository) RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, owner_id, serial, expire, client_cert, client_key, issuing_ca, 
	      ca_chain, private_key_type FROM certs WHERE owner_id = $1 ORDER BY expire LIMIT $2 OFFSET $3;`
	rows, err := cr.db.Query(q, ownerID, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve certs due to %s", err))
		return certs.Page{}, err
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		var dbcrt dbCert
		if err := rows.Scan(&dbcrt.ThingID, &dbcrt.OwnerID, &dbcrt.Serial, &dbcrt.Expire,
			&dbcrt.ClientCert, &dbcrt.ClientKey, &dbcrt.IssuingCA, &dbcrt.CAChain, &dbcrt.PrivateKeyType); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved cert due to %s", err))
			return certs.Page{}, err
		}
		certificates = append(certificates, toCert(dbcrt))
	}

	q = `SELECT COUNT(*) FROM certs WHERE owner_id = $1`
	var total uint64
	if err := cr.db.QueryRow(q, ownerID).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count certs due to %s", err))
		return certs.Page{}, err
	}

	return certs.Page{
		Total: total,
		Certs: certificates,
	}, nil
}

func (cr certsRepository) Save(ctx context.Context, cert certs.Cert) (string, error) {
	q := `INSERT INTO certs (thing_id, owner_id, serial, expire, client_cert, client_key, 
	      issuing_ca, ca_chain, private_key_type) 
	      VALUES (:thing_id, :owner_id, :serial, :expire, :client_cert, :client_key, 
	      :issuing_ca, :ca_chain, :private_key_type)`

	tx, err := cr.db.Beginx()
	if err != nil {
		return "", errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	dbcrt := toDBCert(cert)

	if _, err := tx.NamedExec(q, dbcrt); err != nil {
		e := err
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			e = errors.New("error conflict")
		}

		cr.rollback("Failed to insert a Cert", tx, err)
		return "", errors.Wrap(dbutil.ErrCreateEntity, e)
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Cert save", tx, err)
		return "", errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return cert.Serial, nil
}

func (cr certsRepository) Remove(ctx context.Context, ownerID, serial string) error {
	cert, err := cr.RetrieveBySerial(ctx, ownerID, serial)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	revokeQ := `INSERT INTO revoked_certs (serial, owner_id, thing_id, revoked_at) 
	            VALUES ($1, $2, $3, NOW())`
	if _, err := tx.ExecContext(ctx, revokeQ, serial, ownerID, cert.ThingID); err != nil {
		cr.rollbackTx("Failed to insert into revoked_certs", tx, err)
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	deleteQ := `DELETE FROM certs WHERE serial = $1 AND owner_id = $2`
	if _, err := tx.ExecContext(ctx, deleteQ, serial, ownerID); err != nil {
		cr.rollbackTx("Failed to delete cert", tx, err)
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	if err := tx.Commit(); err != nil {
		cr.rollbackTx("Failed to commit cert removal", tx, err)
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (cr certsRepository) RetrieveRevokedCertificates(ctx context.Context) ([]certs.RevokedCert, error) {
	q := `SELECT serial, revoked_at, thing_id, owner_id FROM revoked_certs ORDER BY revoked_at DESC`
	rows, err := cr.db.QueryContext(ctx, q)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve revoked serials due to %s", err))
		return nil, err
	}
	defer rows.Close()

	var revokedCerts []certs.RevokedCert
	for rows.Next() {
		var cert certs.RevokedCert
		if err := rows.Scan(&cert.Serial, &cert.RevokedAt, &cert.ThingID, &cert.OwnerID); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read revoked serial due to %s", err))
			return nil, err
		}
		revokedCerts = append(revokedCerts, cert)
	}

	if err := rows.Err(); err != nil {
		cr.log.Error(fmt.Sprintf("Error iterating revoked serials: %s", err))
		return nil, err
	}

	return revokedCerts, nil
}

func (cr certsRepository) RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (certs.Page, error) {
	q := `SELECT thing_id, owner_id, serial, expire, client_cert, client_key, issuing_ca, 
	      ca_chain, private_key_type FROM certs 
	      WHERE owner_id = $1 AND thing_id = $2 ORDER BY expire LIMIT $3 OFFSET $4;`
	rows, err := cr.db.Query(q, ownerID, thingID, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve certs due to %s", err))
		return certs.Page{}, err
	}
	defer rows.Close()

	certificates := []certs.Cert{}
	for rows.Next() {
		var dbcrt dbCert
		if err := rows.Scan(&dbcrt.ThingID, &dbcrt.OwnerID, &dbcrt.Serial, &dbcrt.Expire,
			&dbcrt.ClientCert, &dbcrt.ClientKey, &dbcrt.IssuingCA, &dbcrt.CAChain, &dbcrt.PrivateKeyType); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved cert due to %s", err))
			return certs.Page{}, err
		}
		certificates = append(certificates, toCert(dbcrt))
	}

	q = `SELECT COUNT(*) FROM certs WHERE owner_id = $1 AND thing_id = $2`
	var total uint64
	if err := cr.db.QueryRow(q, ownerID, thingID).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count certs due to %s", err))
		return certs.Page{}, err
	}

	return certs.Page{
		Total: total,
		Certs: certificates,
	}, nil
}

func (cr certsRepository) RetrieveBySerial(ctx context.Context, ownerID, serialID string) (certs.Cert, error) {
	q := `SELECT thing_id, owner_id, serial, expire, client_cert, client_key, issuing_ca, 
	      ca_chain, private_key_type FROM certs WHERE owner_id = $1 AND serial = $2`

	var dbcrt dbCert

	err := cr.db.QueryRowContext(ctx, q, ownerID, serialID).Scan(
		&dbcrt.ThingID,
		&dbcrt.OwnerID,
		&dbcrt.Serial,
		&dbcrt.Expire,
		&dbcrt.ClientCert,
		&dbcrt.ClientKey,
		&dbcrt.IssuingCA,
		&dbcrt.CAChain,
		&dbcrt.PrivateKeyType,
	)

	if err != nil {
		pqErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || (ok && pgerrcode.InvalidTextRepresentation == pqErr.Code) {
			return certs.Cert{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return certs.Cert{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toCert(dbcrt), nil
}

func (cr certsRepository) rollback(content string, tx *sqlx.Tx, err error) {
	cr.log.Error(fmt.Sprintf("%s %s", content, err))

	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

func (cr certsRepository) rollbackTx(content string, tx *sqlx.Tx, err error) {
	cr.log.Error(fmt.Sprintf("%s %s", content, err))

	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

type dbCert struct {
	ThingID        string      `db:"thing_id"`
	Serial         string      `db:"serial"`
	Expire         time.Time   `db:"expire"`
	OwnerID        string      `db:"owner_id"`
	ClientCert     string      `db:"client_cert"`
	ClientKey      string      `db:"client_key"`
	IssuingCA      string      `db:"issuing_ca"`
	CAChain        StringArray `db:"ca_chain"`
	PrivateKeyType string      `db:"private_key_type"`
}

func toDBCert(c certs.Cert) dbCert {
	return dbCert{
		ThingID:        c.ThingID,
		OwnerID:        c.OwnerID,
		Serial:         c.Serial,
		Expire:         c.Expire,
		ClientCert:     c.ClientCert,
		ClientKey:      c.ClientKey,
		IssuingCA:      c.IssuingCA,
		CAChain:        StringArray(c.CAChain),
		PrivateKeyType: c.PrivateKeyType,
	}
}

func toCert(cdb dbCert) certs.Cert {
	return certs.Cert{
		OwnerID:        cdb.OwnerID,
		ThingID:        cdb.ThingID,
		Serial:         cdb.Serial,
		Expire:         cdb.Expire,
		ClientCert:     cdb.ClientCert,
		ClientKey:      cdb.ClientKey,
		IssuingCA:      cdb.IssuingCA,
		CAChain:        []string(cdb.CAChain),
		PrivateKeyType: cdb.PrivateKeyType,
	}
}

// NOTE: custom type for PostgreSQL TEXT[] arrays, might delete later
type StringArray []string

func (a *StringArray) Scan(src interface{}) error {
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

	// Handle empty array
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
