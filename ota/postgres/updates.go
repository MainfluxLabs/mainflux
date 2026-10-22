// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ ota.UpdateRepository = (*updateRepository)(nil)

type updateRepository struct {
	db dbutil.Database
}

// NewUpdateRepository instantiates a PostgreSQL implementation of update repository.
func NewUpdateRepository(db dbutil.Database) ota.UpdateRepository {
	return &updateRepository{
		db: db,
	}
}

func (ur updateRepository) Save(ctx context.Context, u ota.Update, thingIDs ...string) error {
	tx, err := ur.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	qFailActive := `UPDATE thing_updates
		SET status = $1, error = $2, updated_at = $3
		WHERE thing_id = ANY($4)
		AND status = ANY($5);`
	if _, err := tx.ExecContext(ctx, qFailActive, ota.StatusFailed, fmt.Sprintf(ota.ErrMsgSuperseded, u.ID), u.CreatedAt, thingIDs, ota.ActiveStatuses); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	qInsUpdate := `INSERT INTO updates (id, group_id, file_name, version, file_size, checksum, description, timeout, created_at)
		VALUES (:id, :group_id, :file_name, :version, :file_size, :checksum, :description, :timeout, :created_at);`
	if _, err := tx.NamedExecContext(ctx, qInsUpdate, toDBUpdate(u)); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	qInsThingUpdates := `INSERT INTO thing_updates (update_id, thing_id, status, updated_at)
		SELECT $1, unnest($2::uuid[]), $3, $4;`
	if _, err := tx.ExecContext(ctx, qInsThingUpdates, u.ID, thingIDs, ota.StatusPending, u.CreatedAt); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ur updateRepository) RetrieveByID(ctx context.Context, id string) (ota.Update, error) {
	q := `SELECT id, group_id, file_name, version, file_size, checksum, description, timeout, created_at
		FROM updates
		WHERE id = $1;`

	var dbu dbUpdate
	if err := ur.db.QueryRowxContext(ctx, q, id).StructScan(&dbu); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return ota.Update{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return ota.Update{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	stats, err := ur.stats(ctx, id)
	if err != nil {
		return ota.Update{}, err
	}

	return toUpdate(dbu, stats[id]), nil
}

func (ur updateRepository) RetrieveByGroup(ctx context.Context, groupID string, pm ota.PageMetadata) (ota.UpdatesPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, ota.UpdateOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	q := fmt.Sprintf(`SELECT id, group_id, file_name, version, file_size, checksum, description, timeout, created_at
		FROM updates
		WHERE group_id = :group_id
		ORDER BY %s %s %s;`, oq, dq, olq)
	qc := `SELECT COUNT(*) FROM updates WHERE group_id = :group_id;`

	params := map[string]any{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := ur.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var dbus []dbUpdate
	var ids []string
	for rows.Next() {
		var dbu dbUpdate
		if err := rows.StructScan(&dbu); err != nil {
			return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		dbus = append(dbus, dbu)
		ids = append(ids, dbu.ID)
	}

	stats, err := ur.stats(ctx, ids...)
	if err != nil {
		return ota.UpdatesPage{}, err
	}

	var items []ota.Update
	for _, dbu := range dbus {
		items = append(items, toUpdate(dbu, stats[dbu.ID]))
	}

	total, err := dbutil.Total(ctx, ur.db, qc, params)
	if err != nil {
		return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return ota.UpdatesPage{
		Total:   total,
		Updates: items,
	}, nil
}

func (ur updateRepository) Remove(ctx context.Context, id string) error {
	q := `DELETE FROM updates WHERE id = :id;`

	if _, err := ur.db.NamedExecContext(ctx, q, map[string]any{"id": id}); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (ur updateRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	q := `DELETE FROM updates WHERE group_id = :group_id;`

	if _, err := ur.db.NamedExecContext(ctx, q, map[string]any{"group_id": groupID}); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (ur updateRepository) RetrieveThingUpdates(ctx context.Context, updateID string, pm ota.PageMetadata) (ota.ThingUpdatesPage, error) {
	if _, err := uuid.FromString(updateID); err != nil {
		return ota.ThingUpdatesPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, ota.ThingUpdateOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	sq := statusQuery(pm.Status)
	whereClause := dbutil.BuildWhereClause("update_id = :update_id", sq)

	q := fmt.Sprintf(`SELECT update_id, thing_id, status, error, reported_version, updated_at
		FROM thing_updates
		%s
		ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM thing_updates %s;`, whereClause)

	params := map[string]any{
		"update_id": updateID,
		"status":    pm.Status,
		"limit":     pm.Limit,
		"offset":    pm.Offset,
	}

	rows, err := ur.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return ota.ThingUpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []ota.ThingUpdate
	for rows.Next() {
		var dbtu dbThingUpdate
		if err := rows.StructScan(&dbtu); err != nil {
			return ota.ThingUpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		items = append(items, toThingUpdate(dbtu))
	}

	total, err := dbutil.Total(ctx, ur.db, qc, params)
	if err != nil {
		return ota.ThingUpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return ota.ThingUpdatesPage{
		Total:        total,
		ThingUpdates: items,
	}, nil
}

func (ur updateRepository) RetrieveThingUpdate(ctx context.Context, updateID, thingID string) (ota.ThingUpdate, error) {
	q := `SELECT update_id, thing_id, status, error, reported_version, updated_at
		FROM thing_updates
		WHERE update_id = $1
		AND thing_id = $2;`

	return ur.retrieveThingUpdate(ctx, q, updateID, thingID)
}

func (ur updateRepository) RetrieveActiveThingUpdate(ctx context.Context, thingID string) (ota.ThingUpdate, error) {
	q := `SELECT t.update_id, t.thing_id, t.status, t.error, t.reported_version, t.updated_at
		FROM thing_updates t
		JOIN updates u ON u.id = t.update_id
		WHERE t.thing_id = $1
		AND t.status = ANY($2)
		ORDER BY u.created_at DESC
		LIMIT 1;`

	return ur.retrieveThingUpdate(ctx, q, thingID, ota.ActiveStatuses)
}

func (ur updateRepository) UpdateThingUpdate(ctx context.Context, tu ota.ThingUpdate) error {
	q := `UPDATE thing_updates
		SET status = :status, error = :error, reported_version = :reported_version, updated_at = :updated_at
		WHERE update_id = :update_id
		AND thing_id = :thing_id;`

	res, err := ur.db.NamedExecContext(ctx, q, toDBThingUpdate(tu))
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.StringDataRightTruncationDataException {
			return errors.Wrap(dbutil.ErrMalformedEntity, err)
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

func (ur updateRepository) RetryFailed(ctx context.Context, updateID string) ([]string, error) {
	q := `UPDATE thing_updates
		SET status = :pending, error = '', updated_at = :updated_at
		WHERE update_id = :update_id
		AND status = :failed
		RETURNING thing_id;`

	params := map[string]any{
		"pending":    ota.StatusPending,
		"failed":     ota.StatusFailed,
		"updated_at": time.Now().UTC(),
		"update_id":  updateID,
	}

	rows, err := ur.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}
	defer rows.Close()

	var thingIDs []string
	for rows.Next() {
		var thingID string
		if err := rows.Scan(&thingID); err != nil {
			return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
		}
		thingIDs = append(thingIDs, thingID)
	}

	return thingIDs, nil
}

func (ur updateRepository) FailTimedOut(ctx context.Context, now time.Time) error {
	q := `UPDATE thing_updates t
		SET status = :failed, error = :error, updated_at = :now
		FROM updates u
		WHERE u.id = t.update_id
		AND t.status = ANY(:active)
		AND t.updated_at + u.timeout * INTERVAL '1 second' < :now;`

	params := map[string]any{
		"failed": ota.StatusFailed,
		"error":  ota.ErrMsgTimedOut,
		"now":    now,
		"active": ota.ActiveStatuses,
	}

	if _, err := ur.db.NamedExecContext(ctx, q, params); err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

func (ur updateRepository) RemoveThingUpdatesByThing(ctx context.Context, thingID string) error {
	q := `DELETE FROM thing_updates WHERE thing_id = :thing_id;`

	if _, err := ur.db.NamedExecContext(ctx, q, map[string]any{"thing_id": thingID}); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (ur updateRepository) retrieveThingUpdate(ctx context.Context, q string, args ...any) (ota.ThingUpdate, error) {
	var dbtu dbThingUpdate
	if err := ur.db.QueryRowxContext(ctx, q, args...).StructScan(&dbtu); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return ota.ThingUpdate{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return ota.ThingUpdate{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toThingUpdate(dbtu), nil
}

// stats returns thing update counts by status for each of the given updates.
func (ur updateRepository) stats(ctx context.Context, updateIDs ...string) (map[string]ota.UpdateStats, error) {
	q := `SELECT update_id,
		COUNT(*) AS targets,
		COUNT(*) FILTER (WHERE status = 'pending') AS pending,
		COUNT(*) FILTER (WHERE status = 'downloading') AS downloading,
		COUNT(*) FILTER (WHERE status = 'installing') AS installing,
		COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded,
		COUNT(*) FILTER (WHERE status = 'failed') AS failed
		FROM thing_updates
		WHERE update_id = ANY($1)
		GROUP BY update_id;`

	var rows []dbUpdateStats
	if err := ur.db.SelectContext(ctx, &rows, q, updateIDs); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	stats := make(map[string]ota.UpdateStats, len(rows))
	for _, r := range rows {
		stats[r.UpdateID] = ota.UpdateStats{
			Targets:     r.Targets,
			Pending:     r.Pending,
			Downloading: r.Downloading,
			Installing:  r.Installing,
			Succeeded:   r.Succeeded,
			Failed:      r.Failed,
		}
	}

	return stats, nil
}

func statusQuery(status string) string {
	if status == "" {
		return ""
	}
	return "status = :status"
}

func createErr(err error) error {
	pgErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pgErr.Code {
		case pgerrcode.InvalidTextRepresentation, pgerrcode.StringDataRightTruncationDataException:
			return errors.Wrap(dbutil.ErrMalformedEntity, err)
		case pgerrcode.UniqueViolation:
			return errors.Wrap(dbutil.ErrConflict, err)
		}
	}

	return errors.Wrap(dbutil.ErrCreateEntity, err)
}

type dbUpdate struct {
	ID          string    `db:"id"`
	GroupID     string    `db:"group_id"`
	FileName    string    `db:"file_name"`
	Version     string    `db:"version"`
	FileSize    int64     `db:"file_size"`
	Checksum    string    `db:"checksum"`
	Description string    `db:"description"`
	Timeout     int64     `db:"timeout"`
	CreatedAt   time.Time `db:"created_at"`
}

type dbUpdateStats struct {
	UpdateID    string `db:"update_id"`
	Targets     uint64 `db:"targets"`
	Pending     uint64 `db:"pending"`
	Downloading uint64 `db:"downloading"`
	Installing  uint64 `db:"installing"`
	Succeeded   uint64 `db:"succeeded"`
	Failed      uint64 `db:"failed"`
}

func toDBUpdate(u ota.Update) dbUpdate {
	return dbUpdate{
		ID:          u.ID,
		GroupID:     u.GroupID,
		FileName:    u.FileName,
		Version:     u.Version,
		FileSize:    u.FileSize,
		Checksum:    u.Checksum,
		Description: u.Description,
		Timeout:     int64(u.Timeout / time.Second),
		CreatedAt:   u.CreatedAt,
	}
}

func toUpdate(dbu dbUpdate, stats ota.UpdateStats) ota.Update {
	return ota.Update{
		ID:          dbu.ID,
		GroupID:     dbu.GroupID,
		FileName:    dbu.FileName,
		Version:     dbu.Version,
		FileSize:    dbu.FileSize,
		Checksum:    dbu.Checksum,
		Description: dbu.Description,
		Timeout:     time.Duration(dbu.Timeout) * time.Second,
		CreatedAt:   dbu.CreatedAt,
		Stats:       stats,
	}
}

type dbThingUpdate struct {
	UpdateID        string    `db:"update_id"`
	ThingID         string    `db:"thing_id"`
	Status          string    `db:"status"`
	Error           string    `db:"error"`
	ReportedVersion string    `db:"reported_version"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func toDBThingUpdate(tu ota.ThingUpdate) dbThingUpdate {
	return dbThingUpdate{
		UpdateID:        tu.UpdateID,
		ThingID:         tu.ThingID,
		Status:          tu.Status,
		Error:           tu.Error,
		ReportedVersion: tu.ReportedVersion,
		UpdatedAt:       tu.UpdatedAt,
	}
}

func toThingUpdate(dbtu dbThingUpdate) ota.ThingUpdate {
	return ota.ThingUpdate{
		UpdateID:        dbtu.UpdateID,
		ThingID:         dbtu.ThingID,
		Status:          dbtu.Status,
		Error:           dbtu.Error,
		ReportedVersion: dbtu.ReportedVersion,
		UpdatedAt:       dbtu.UpdatedAt.UTC(),
	}
}
