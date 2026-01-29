// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.ProfileRepository = (*profileRepository)(nil)

type profileRepository struct {
	db dbutil.Database
}

// NewProfileRepository instantiates a PostgreSQL implementation of profile
// repository.
func NewProfileRepository(db dbutil.Database) things.ProfileRepository {
	return &profileRepository{
		db: db,
	}
}

func (pr profileRepository) Save(ctx context.Context, profiles ...things.Profile) ([]things.Profile, error) {
	tx, err := pr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	defer tx.Rollback()

	q := `INSERT INTO profiles (id, group_id, name, metadata, config)
		  VALUES (:id, :group_id, :name, :metadata, :config);`

	for _, profile := range profiles {
		dbpr := toDBProfile(profile)

		_, err = tx.NamedExecContext(ctx, q, dbpr)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Profile{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Profile{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Profile{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}
			return []things.Profile{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Profile{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return profiles, nil
}

func (pr profileRepository) Update(ctx context.Context, profile things.Profile) error {
	q := `UPDATE profiles SET name = :name, metadata = :metadata, config = :config WHERE id = :id;`

	dbpr := toDBProfile(profile)

	res, err := pr.db.NamedExecContext(ctx, q, dbpr)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return dbutil.ErrNotFound
	}

	return nil
}

func (pr profileRepository) RetrieveByID(ctx context.Context, id string) (things.Profile, error) {
	q := `SELECT group_id, name, metadata, config FROM profiles WHERE id = $1;`

	dbpr := dbProfile{
		ID: id,
	}

	if err := pr.db.QueryRowxContext(ctx, q, id).StructScan(&dbpr); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Profile{}, dbutil.ErrNotFound
		}
		return things.Profile{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toProfile(dbpr), nil
}

func (pr profileRepository) BackupAll(ctx context.Context) ([]things.Profile, error) {
	query := "SELECT id, group_id, name, metadata, config FROM profiles"

	var items []dbProfile
	err := pr.db.SelectContext(ctx, &items, query)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var profiles []things.Profile
	for _, i := range items {
		profiles = append(profiles, toProfile(i))
	}

	return profiles, nil
}

func (pr profileRepository) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, name, metadata, config FROM profiles %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM profiles %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return pr.retrieve(ctx, query, cquery, params)
}

func (pr profileRepository) RetrieveByThing(ctx context.Context, thID string) (things.Profile, error) {
	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thID); err != nil {
		return things.Profile{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	q := `SELECT pr.id, pr.group_id, pr.name, pr.metadata, pr.config
		FROM things ths, profiles pr
		WHERE ths.profile_id = pr.id and ths.id = :thing;`
	params := map[string]any{
		"thing": thID,
	}

	rows, err := pr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Profile{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var item things.Profile
	for rows.Next() {
		dbpr := dbProfile{}
		if err := rows.StructScan(&dbpr); err != nil {
			return things.Profile{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		item = toProfile(dbpr)
	}

	return item, nil
}

func (pr profileRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbpr := dbProfile{
			ID: id,
		}
		q := `DELETE FROM profiles WHERE id = :id`
		_, err := pr.db.NamedExecContext(ctx, q, dbpr)
		if err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				if pgErr.Code == pgerrcode.ForeignKeyViolation {
					return errors.Wrap(things.ErrProfileAssigned, err)
				}
			}

			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (pr profileRepository) RetrieveByGroups(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	if len(groupIDs) == 0 {
		return things.ProfilesPage{}, nil
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	giq := dbutil.GetGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	m, mq, err := dbutil.GetMetadataQuery(pm.Metadata)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	whereClause := dbutil.BuildWhereClause(giq, nq, mq)
	query := fmt.Sprintf(`SELECT id, group_id, name, metadata, config FROM profiles %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM profiles %s;`, whereClause)

	params := map[string]any{
		"name":     name,
		"metadata": m,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return pr.retrieve(ctx, query, cquery, params)
}

func (pr profileRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (things.ProfilesPage, error) {
	rows, err := pr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.Profile
	for rows.Next() {
		dbpr := dbProfile{}
		if err := rows.StructScan(&dbpr); err != nil {
			return things.ProfilesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		pr := toProfile(dbpr)

		items = append(items, pr)
	}

	total, err := dbutil.Total(ctx, pr.db, cquery, params)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := things.ProfilesPage{
		Profiles: items,
		Total:    total,
	}

	return page, nil
}

// dbJSONB type for handling JSONB data properly in database/sql.
type dbJSONB map[string]any

// Scan implements the database/sql scanner interface.
// When interface is nil `m` is set to nil.
// If error occurs on casting data then m points to empty metadata.
func (m *dbJSONB) Scan(value any) error {
	if value == nil {
		m = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		m = &dbJSONB{}
		return dbutil.ErrScanMetadata
	}

	if err := json.Unmarshal(b, m); err != nil {
		return err
	}

	return nil
}

// Value implements database/sql valuer interface.
func (m dbJSONB) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, err
}

type dbProfile struct {
	ID       string  `db:"id"`
	GroupID  string  `db:"group_id"`
	Name     string  `db:"name"`
	Config   dbJSONB `db:"config"`
	Metadata dbJSONB `db:"metadata"`
}

func toDBProfile(pr things.Profile) dbProfile {
	return dbProfile{
		ID:       pr.ID,
		GroupID:  pr.GroupID,
		Name:     pr.Name,
		Config:   pr.Config,
		Metadata: pr.Metadata,
	}
}

func toProfile(pr dbProfile) things.Profile {
	return things.Profile{
		ID:       pr.ID,
		GroupID:  pr.GroupID,
		Name:     pr.Name,
		Config:   pr.Config,
		Metadata: pr.Metadata,
	}
}

func getIDsQuery(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("id IN ('%s') ", strings.Join(ids, "','"))
}
