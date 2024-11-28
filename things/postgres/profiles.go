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

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.ProfileRepository = (*profileRepository)(nil)

type profileRepository struct {
	db Database
}

// NewProfileRepository instantiates a PostgreSQL implementation of profile
// repository.
func NewProfileRepository(db Database) things.ProfileRepository {
	return &profileRepository{
		db: db,
	}
}

func (cr profileRepository) Save(ctx context.Context, profiles ...things.Profile) ([]things.Profile, error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO profiles (id, group_id, name, metadata, config)
		  VALUES (:id, :group_id, :name, :metadata, :config);`

	for _, profile := range profiles {
		dbpr := toDBProfile(profile)

		_, err = tx.NamedExecContext(ctx, q, dbpr)
		if err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Profile{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Profile{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Profile{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}
			return []things.Profile{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Profile{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return profiles, nil
}

func (cr profileRepository) Update(ctx context.Context, profile things.Profile) error {
	q := `UPDATE profiles SET name = :name, metadata = :metadata, config = :config WHERE id = :id;`

	dbpr := toDBProfile(profile)

	res, err := cr.db.NamedExecContext(ctx, q, dbpr)
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			case pgerrcode.StringDataRightTruncationDataException:
				return errors.Wrap(errors.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (cr profileRepository) RetrieveByID(ctx context.Context, id string) (things.Profile, error) {
	q := `SELECT group_id, name, metadata, config FROM profiles WHERE id = $1;`

	dbpr := dbProfile{
		ID: id,
	}

	if err := cr.db.QueryRowxContext(ctx, q, id).StructScan(&dbpr); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Profile{}, errors.ErrNotFound
		}
		return things.Profile{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toProfile(dbpr), nil
}

func (cr profileRepository) RetrieveAll(ctx context.Context) ([]things.Profile, error) {
	prPage, err := cr.retrieve(ctx, []string{}, true, things.PageMetadata{})
	if err != nil {
		return []things.Profile{}, err
	}

	return prPage.Profiles, nil
}

func (cr profileRepository) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ProfilesPage, error) {
	return cr.retrieve(ctx, []string{}, false, pm)
}

func (cr profileRepository) RetrieveByThing(ctx context.Context, thID string) (things.Profile, error) {
	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thID); err != nil {
		return things.Profile{}, errors.Wrap(errors.ErrNotFound, err)
	}

	var q string
	q = fmt.Sprintf(`SELECT pr.id, pr.group_id, pr.name, pr.metadata, pr.config
				FROM things ths, profiles pr
				WHERE ths.profile_id = pr.id and ths.id = :thing;`)
	params := map[string]interface{}{
		"thing": thID,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Profile{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var item things.Profile
	for rows.Next() {
		dbpr := dbProfile{}
		if err := rows.StructScan(&dbpr); err != nil {
			return things.Profile{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		item = toProfile(dbpr)
	}

	return item, nil
}

func (cr profileRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbpr := dbProfile{
			ID: id,
		}
		q := `DELETE FROM profiles WHERE id = :id`
		_, err := cr.db.NamedExecContext(ctx, q, dbpr)
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (cr profileRepository) RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm things.PageMetadata) (things.ProfilesPage, error) {
	if len(groupIDs) == 0 {
		return things.ProfilesPage{}, nil
	}

	prPage, err := cr.retrieve(ctx, groupIDs, false, pm)
	if err != nil {
		return things.ProfilesPage{}, err
	}

	return prPage, nil
}

func (cr profileRepository) retrieve(ctx context.Context, groupIDs []string, allRows bool, pm things.PageMetadata) (things.ProfilesPage, error) {
	idsq := getGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	meta, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	var query []string
	if idsq != "" {
		query = append(query, idsq)
	}
	if mq != "" {
		query = append(query, mq)
	}
	if nq != "" {
		query = append(query, nq)
	}

	var whereClause string
	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT id, group_id, name, metadata, config FROM profiles %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)

	if allRows {
		q = "SELECT id, group_id, name, metadata, config FROM profiles"
	}

	params := map[string]interface{}{
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": meta,
	}
	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items := []things.Profile{}
	for rows.Next() {
		dbpr := dbProfile{}
		if err := rows.StructScan(&dbpr); err != nil {
			return things.ProfilesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		pr := toProfile(dbpr)

		items = append(items, pr)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM profiles %s;`, whereClause)

	total, err := total(ctx, cr.db, cq, params)
	if err != nil {
		return things.ProfilesPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.ProfilesPage{
		Profiles: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return page, nil
}

// dbJSONB type for handling JSONB data properly in database/sql.
type dbJSONB map[string]interface{}

// Scan implements the database/sql scanner interface.
// When interface is nil `m` is set to nil.
// If error occurs on casting data then m points to empty metadata.
func (m *dbJSONB) Scan(value interface{}) error {
	if value == nil {
		m = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		m = &dbJSONB{}
		return errors.ErrScanMetadata
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

func getGroupIDsQuery(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("group_id IN ('%s') ", strings.Join(ids, "','"))
}

func getIDsQuery(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("id IN ('%s') ", strings.Join(ids, "','"))
}

func total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}
