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
	"github.com/jmoiron/sqlx"
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

	qFailActive := `WITH old AS (
			SELECT update_id, thing_id, status FROM thing_updates
			WHERE thing_id = ANY(:thing_ids)
			AND status = ANY(:active)
			FOR UPDATE
		)
		UPDATE thing_updates t
		SET status = :failed, error = :error, updated_at = :updated_at
		FROM old
		WHERE t.update_id = old.update_id
		AND t.thing_id = old.thing_id
		RETURNING t.update_id, old.status;`
	failParams := map[string]any{
		"failed":     ota.StatusFailed,
		"error":      fmt.Sprintf(ota.ErrMsgSuperseded, u.ID),
		"updated_at": u.CreatedAt,
		"thing_ids":  thingIDs,
		"active":     ota.ActiveStatuses,
	}
	var oldStatuses []dbOldStatus
	if err := namedSelect(ctx, tx, &oldStatuses, qFailActive, failParams); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	qInsUpdate := `INSERT INTO updates (id, group_id, file_name, version, file_size, checksum,
		description, timeout, created_at, status, targets, pending_count, downloading_count,
		installing_count, succeeded_count, failed_count)
		VALUES (:id, :group_id, :file_name, :version, :file_size, :checksum,
		:description, :timeout, :created_at, :status, :targets, :pending_count, :downloading_count,
		:installing_count, :succeeded_count, :failed_count);`
	if _, err := tx.NamedExecContext(ctx, qInsUpdate, toDBUpdate(u)); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	qInsThingUpdates := `INSERT INTO thing_updates (update_id, thing_id, status, updated_at)
		SELECT :update_id, unnest(CAST(:thing_ids AS uuid[])), :pending, :updated_at;`
	insParams := map[string]any{
		"update_id":  u.ID,
		"thing_ids":  thingIDs,
		"pending":    ota.StatusPending,
		"updated_at": u.CreatedAt,
	}
	if _, err := tx.NamedExecContext(ctx, qInsThingUpdates, insParams); err != nil {
		tx.Rollback()
		return createErr(err)
	}

	if err := adjustStats(ctx, tx, oldStatuses, ota.StatusFailed); err != nil {
		tx.Rollback()
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (ur updateRepository) RetrieveByID(ctx context.Context, id string) (ota.Update, error) {
	q := `SELECT id, group_id, file_name, version, file_size, checksum, description,
		timeout, created_at, status, targets, pending_count, downloading_count, installing_count,
		succeeded_count, failed_count
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

	return toUpdate(dbu), nil
}

func (ur updateRepository) RetrieveByGroup(ctx context.Context, groupID string, pm ota.PageMetadata) (ota.UpdatesPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, ota.UpdateOrderFields)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	sq := statusQuery(pm.Status)
	whereClause := dbutil.BuildWhereClause("group_id = :group_id", sq)

	q := fmt.Sprintf(`SELECT id, group_id, file_name, version, file_size, checksum, description,
		timeout, created_at, status, targets, pending_count, downloading_count, installing_count,
		succeeded_count, failed_count
		FROM updates
		%s
		ORDER BY %s %s %s;`, whereClause, oq, dq, olq)
	qc := fmt.Sprintf(`SELECT COUNT(*) FROM updates %s;`, whereClause)

	params := map[string]any{
		"group_id": groupID,
		"status":   pm.Status,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := ur.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []ota.Update
	for rows.Next() {
		var dbu dbUpdate
		if err := rows.StructScan(&dbu); err != nil {
			return ota.UpdatesPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		items = append(items, toUpdate(dbu))
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
	q := `WITH old AS (
			SELECT update_id, thing_id, status FROM thing_updates
			WHERE update_id = :update_id
			AND thing_id = :thing_id
			AND status = ANY(:active)
			FOR UPDATE
		)
		UPDATE thing_updates t
		SET status = :status, error = :error,
		reported_version = :reported_version, updated_at = :updated_at
		FROM old
		WHERE t.update_id = old.update_id
		AND t.thing_id = old.thing_id
		RETURNING t.update_id, old.status;`

	params := map[string]any{
		"status":           tu.Status,
		"error":            tu.Error,
		"reported_version": tu.ReportedVersion,
		"updated_at":       tu.UpdatedAt,
		"update_id":        tu.UpdateID,
		"thing_id":         tu.ThingID,
		"active":           ota.ActiveStatuses,
	}

	tx, err := ur.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	var oldStatuses []dbOldStatus
	if err := namedSelect(ctx, tx, &oldStatuses, q, params); err != nil {
		tx.Rollback()
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.StringDataRightTruncationDataException {
			return errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	if len(oldStatuses) == 0 {
		tx.Rollback()
		return dbutil.ErrNotFound
	}

	if err := adjustStats(ctx, tx, oldStatuses, tu.Status); err != nil {
		tx.Rollback()
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return nil
}

func (ur updateRepository) RetrieveFailedThingIDs(ctx context.Context, updateID string) ([]string, error) {
	q := `SELECT thing_id FROM thing_updates WHERE update_id = $1 AND status = $2;`

	var thingIDs []string
	if err := ur.db.SelectContext(ctx, &thingIDs, q, updateID, ota.StatusFailed); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return thingIDs, nil
}

func (ur updateRepository) RetryFailed(ctx context.Context, updateID string, thingIDs []string) ([]string, error) {
	// Things with a newer update are skipped.
	q := `UPDATE thing_updates t
		SET status = :pending, error = '', updated_at = :updated_at
		FROM updates u
		WHERE u.id = t.update_id
		AND t.update_id = :update_id
		AND t.thing_id = ANY(:thing_ids)
		AND t.status = :failed
		AND NOT EXISTS (
			SELECT 1 FROM thing_updates nt
			JOIN updates nu ON nu.id = nt.update_id
			WHERE nt.thing_id = t.thing_id
			AND nu.created_at > u.created_at
		)
		RETURNING t.thing_id;`

	params := map[string]any{
		"pending":    ota.StatusPending,
		"failed":     ota.StatusFailed,
		"updated_at": time.Now().UTC(),
		"update_id":  updateID,
		"thing_ids":  thingIDs,
	}

	tx, err := ur.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	var retriedThingIDs []string
	if err := namedSelect(ctx, tx, &retriedThingIDs, q, params); err != nil {
		tx.Rollback()
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	oldStatuses := make([]dbOldStatus, len(retriedThingIDs))
	for i := range oldStatuses {
		oldStatuses[i] = dbOldStatus{UpdateID: updateID, Status: ota.StatusFailed}
	}

	if err := adjustStats(ctx, tx, oldStatuses, ota.StatusPending); err != nil {
		tx.Rollback()
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	return retriedThingIDs, nil
}

func (ur updateRepository) FailTimedOut(ctx context.Context, now time.Time) error {
	q := `WITH old AS (
			SELECT t.update_id, t.thing_id, t.status FROM thing_updates t
			JOIN updates u ON u.id = t.update_id
			WHERE t.status = ANY(:active)
			AND t.updated_at + u.timeout * INTERVAL '1 second' < :now
			FOR UPDATE OF t
		)
		UPDATE thing_updates t
		SET status = :failed, error = :error, updated_at = :now
		FROM old
		WHERE t.update_id = old.update_id
		AND t.thing_id = old.thing_id
		RETURNING t.update_id, old.status;`

	params := map[string]any{
		"failed": ota.StatusFailed,
		"error":  ota.ErrMsgTimedOut,
		"now":    now,
		"active": ota.ActiveStatuses,
	}

	return ur.changeThingUpdates(ctx, q, params, ota.StatusFailed, dbutil.ErrUpdateEntity)
}

func (ur updateRepository) RemoveThingUpdatesByThing(ctx context.Context, thingID string) error {
	q := `DELETE FROM thing_updates
		WHERE thing_id = :thing_id
		RETURNING update_id, status;`

	params := map[string]any{"thing_id": thingID}

	return ur.changeThingUpdates(ctx, q, params, "", dbutil.ErrRemoveEntity)
}

// changeThingUpdates runs q, which returns the old statuses of the thing updates it changed to
// newStatus, and applies the changes to their updates in the same transaction.
func (ur updateRepository) changeThingUpdates(ctx context.Context, q string, params map[string]any, newStatus string, errType error) error {
	tx, err := ur.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errType, err)
	}

	var oldStatuses []dbOldStatus
	if err := namedSelect(ctx, tx, &oldStatuses, q, params); err != nil {
		tx.Rollback()
		return errors.Wrap(errType, err)
	}

	if err := adjustStats(ctx, tx, oldStatuses, newStatus); err != nil {
		tx.Rollback()
		return errors.Wrap(errType, err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(errType, err)
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

// adjustStats moves each changed thing update from its old status counter to the newStatus
// counter in its update's stats, and stores the stats with the resulting status. An empty
// newStatus means the thing updates were removed. Every change of thing updates calls it in the
// same transaction, so the stored stats and status never drift.
func adjustStats(ctx context.Context, tx *sqlx.Tx, oldStatuses []dbOldStatus, newStatus string) error {
	if len(oldStatuses) == 0 {
		return nil
	}

	updateIDs := make([]string, len(oldStatuses))
	for i, old := range oldStatuses {
		updateIDs[i] = old.UpdateID
	}

	// The lock orders changes of the same update, so each one counts into the latest stats.
	qLock := `SELECT id, targets, pending_count, downloading_count, installing_count,
		succeeded_count, failed_count
		FROM updates
		WHERE id = ANY(:update_ids)
		ORDER BY id
		FOR UPDATE;`
	qStore := `UPDATE updates u
		SET status = s.status, targets = s.targets, pending_count = s.pending_count,
		downloading_count = s.downloading_count, installing_count = s.installing_count,
		succeeded_count = s.succeeded_count, failed_count = s.failed_count
		FROM unnest(CAST(:update_ids AS uuid[]), CAST(:statuses AS varchar[]),
			CAST(:targets AS bigint[]), CAST(:pending_counts AS bigint[]),
			CAST(:downloading_counts AS bigint[]), CAST(:installing_counts AS bigint[]),
			CAST(:succeeded_counts AS bigint[]), CAST(:failed_counts AS bigint[]))
			AS s (id, status, targets, pending_count, downloading_count, installing_count,
			succeeded_count, failed_count)
		WHERE u.id = s.id;`

	var dbus []dbUpdate
	if err := namedSelect(ctx, tx, &dbus, qLock, map[string]any{"update_ids": updateIDs}); err != nil {
		return err
	}

	updates := make(map[string]ota.Update, len(dbus))
	for _, dbu := range dbus {
		updates[dbu.ID] = toUpdate(dbu)
	}

	for _, old := range oldStatuses {
		u, ok := updates[old.UpdateID]
		if !ok {
			return dbutil.ErrNotFound
		}
		u.Stats.Remove(old.Status)
		if newStatus != "" {
			u.Stats.Add(newStatus)
		}
		updates[old.UpdateID] = u
	}

	if _, err := tx.NamedExecContext(ctx, qStore, toDBUpdateStats(updates)); err != nil {
		return err
	}

	return nil
}

// namedSelect runs a named query in the transaction and scans its rows into dest.
func namedSelect(ctx context.Context, tx *sqlx.Tx, dest any, q string, params map[string]any) error {
	query, args, err := tx.BindNamed(q, params)
	if err != nil {
		return err
	}

	return tx.SelectContext(ctx, dest, query, args...)
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
	ID               string    `db:"id"`
	GroupID          string    `db:"group_id"`
	FileName         string    `db:"file_name"`
	Version          string    `db:"version"`
	FileSize         int64     `db:"file_size"`
	Checksum         string    `db:"checksum"`
	Description      string    `db:"description"`
	Timeout          int64     `db:"timeout"`
	CreatedAt        time.Time `db:"created_at"`
	Status           string    `db:"status"`
	Targets          uint64    `db:"targets"`
	PendingCount     uint64    `db:"pending_count"`
	DownloadingCount uint64    `db:"downloading_count"`
	InstallingCount  uint64    `db:"installing_count"`
	SucceededCount   uint64    `db:"succeeded_count"`
	FailedCount      uint64    `db:"failed_count"`
}

func toDBUpdate(u ota.Update) dbUpdate {
	return dbUpdate{
		ID:               u.ID,
		GroupID:          u.GroupID,
		FileName:         u.FileName,
		Version:          u.Version,
		FileSize:         u.FileSize,
		Checksum:         u.Checksum,
		Description:      u.Description,
		Timeout:          int64(u.Timeout / time.Second),
		CreatedAt:        u.CreatedAt,
		Status:           u.Status,
		Targets:          u.Stats.Targets,
		PendingCount:     u.Stats.Pending,
		DownloadingCount: u.Stats.Downloading,
		InstallingCount:  u.Stats.Installing,
		SucceededCount:   u.Stats.Succeeded,
		FailedCount:      u.Stats.Failed,
	}
}

func toUpdate(dbu dbUpdate) ota.Update {
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
		Status:      dbu.Status,
		Stats: ota.UpdateStats{
			Targets:     dbu.Targets,
			Pending:     dbu.PendingCount,
			Downloading: dbu.DownloadingCount,
			Installing:  dbu.InstallingCount,
			Succeeded:   dbu.SucceededCount,
			Failed:      dbu.FailedCount,
		},
	}
}

// dbUpdateStats holds the stats and status of several updates as parallel columns for unnest.
type dbUpdateStats struct {
	UpdateIDs         []string `db:"update_ids"`
	Statuses          []string `db:"statuses"`
	Targets           []uint64 `db:"targets"`
	PendingCounts     []uint64 `db:"pending_counts"`
	DownloadingCounts []uint64 `db:"downloading_counts"`
	InstallingCounts  []uint64 `db:"installing_counts"`
	SucceededCounts   []uint64 `db:"succeeded_counts"`
	FailedCounts      []uint64 `db:"failed_counts"`
}

func toDBUpdateStats(updates map[string]ota.Update) dbUpdateStats {
	var dbus dbUpdateStats
	for _, u := range updates {
		dbus.UpdateIDs = append(dbus.UpdateIDs, u.ID)
		dbus.Statuses = append(dbus.Statuses, u.Stats.Status())
		dbus.Targets = append(dbus.Targets, u.Stats.Targets)
		dbus.PendingCounts = append(dbus.PendingCounts, u.Stats.Pending)
		dbus.DownloadingCounts = append(dbus.DownloadingCounts, u.Stats.Downloading)
		dbus.InstallingCounts = append(dbus.InstallingCounts, u.Stats.Installing)
		dbus.SucceededCounts = append(dbus.SucceededCounts, u.Stats.Succeeded)
		dbus.FailedCounts = append(dbus.FailedCounts, u.Stats.Failed)
	}

	return dbus
}

// dbOldStatus is the status a thing update of an update had before it changed.
type dbOldStatus struct {
	UpdateID string `db:"update_id"`
	Status   string `db:"status"`
}

type dbThingUpdate struct {
	UpdateID        string    `db:"update_id"`
	ThingID         string    `db:"thing_id"`
	Status          string    `db:"status"`
	Error           string    `db:"error"`
	ReportedVersion string    `db:"reported_version"`
	UpdatedAt       time.Time `db:"updated_at"`
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
