package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ things.GroupMembershipsRepository = (*groupMembershipsRepository)(nil)

type groupMembershipsRepository struct {
	db dbutil.Database
}

// NewGroupMembershipsRepository instantiates a PostgreSQL implementation of membership repository.
func NewGroupMembershipsRepository(db dbutil.Database) things.GroupMembershipsRepository {
	return &groupMembershipsRepository{
		db: db,
	}
}

func (mr groupMembershipsRepository) Save(ctx context.Context, gms ...things.GroupMembership) error {
	tx, err := mr.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := `INSERT INTO group_memberships (member_id, group_id, role) VALUES (:member_id, :group_id, :role);`

	for _, g := range gms {
		dbgm := toDBGroupMembership(g)
		if _, err := mr.db.NamedExecContext(ctx, q, dbgm); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(dbutil.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(things.ErrGroupMembershipExists, errors.New(pgErr.Detail))
				}
			}
			return errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	return nil
}

func (mr groupMembershipsRepository) RetrieveRole(ctx context.Context, gm things.GroupMembership) (string, error) {
	q := `SELECT role FROM group_memberships WHERE member_id = $1 AND group_id = $2;`

	var role string
	if err := mr.db.QueryRowxContext(ctx, q, gm.MemberID, gm.GroupID).Scan(&role); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return "", errors.Wrap(dbutil.ErrNotFound, err)
		}

		return "", errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return role, nil
}

func (mr groupMembershipsRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembershipsPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	q := fmt.Sprintf(`SELECT member_id, role FROM group_memberships WHERE group_id = :group_id %s;`, olq)

	params := map[string]any{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupMembershipsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMembership
	for rows.Next() {
		dbgm := dbGroupMembership{}
		if err := rows.StructScan(&dbgm); err != nil {
			return things.GroupMembershipsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		gm := toGroupMemberships(dbgm)
		items = append(items, gm)
	}

	cq := `SELECT COUNT(*) FROM group_memberships WHERE group_id = :group_id;`

	total, err := dbutil.Total(ctx, mr.db, cq, params)
	if err != nil {
		return things.GroupMembershipsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := things.GroupMembershipsPage{
		GroupMemberships: items,
		Total:            total,
	}

	return page, nil
}

func (mr groupMembershipsRepository) BackupAll(ctx context.Context) ([]things.GroupMembership, error) {
	q := `SELECT member_id, group_id, role FROM group_memberships;`

	rows, err := mr.db.NamedQueryContext(ctx, q, map[string]any{})
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMembership
	for rows.Next() {
		dbgm := dbGroupMembership{}
		if err := rows.StructScan(&dbgm); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		gm := toGroupMemberships(dbgm)
		items = append(items, gm)
	}

	return items, nil
}

func (mr groupMembershipsRepository) BackupByGroup(ctx context.Context, groupID string) ([]things.GroupMembership, error) {
	q := `SELECT member_id, group_id, role FROM group_memberships WHERE group_id = :group_id;`

	rows, err := mr.db.NamedQueryContext(ctx, q, map[string]any{
		"group_id": groupID,
	})
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMembership
	for rows.Next() {
		dbgm := dbGroupMembership{}
		if err := rows.StructScan(&dbgm); err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		gm := toGroupMemberships(dbgm)
		items = append(items, gm)
	}

	return items, nil
}

func (mr groupMembershipsRepository) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	var groupIDs []string

	q := `SELECT group_id FROM group_memberships WHERE member_id = $1;`

	if err := mr.db.SelectContext(ctx, &groupIDs, q, memberID); err != nil {
		return nil, err
	}

	return groupIDs, nil
}

func (mr groupMembershipsRepository) Remove(ctx context.Context, groupID string, memberIDs ...string) error {
	q := `DELETE FROM group_memberships WHERE member_id = :member_id AND group_id = :group_id;`

	for _, memberID := range memberIDs {
		dbgm := dbGroupMembership{
			MemberID: memberID,
			GroupID:  groupID,
		}

		if _, err := mr.db.NamedExecContext(ctx, q, dbgm); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (mr groupMembershipsRepository) Update(ctx context.Context, gms ...things.GroupMembership) error {
	q := `UPDATE group_memberships SET role = :role WHERE member_id = :member_id AND group_id = :group_id;`

	for _, g := range gms {
		dbgm := toDBGroupMembership(g)
		row, err := mr.db.NamedExecContext(ctx, q, dbgm)
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

		cnt, err := row.RowsAffected()
		if err != nil {
			return errors.Wrap(dbutil.ErrUpdateEntity, err)
		}

		if cnt != 1 {
			return errors.Wrap(dbutil.ErrNotFound, err)
		}
	}

	return nil
}

type dbGroupMembership struct {
	MemberID string `db:"member_id"`
	GroupID  string `db:"group_id"`
	Role     string `db:"role"`
}

func toDBGroupMembership(gm things.GroupMembership) dbGroupMembership {
	return dbGroupMembership{
		MemberID: gm.MemberID,
		GroupID:  gm.GroupID,
		Role:     gm.Role,
	}
}

func toGroupMemberships(dbgm dbGroupMembership) things.GroupMembership {
	return things.GroupMembership{
		GroupID:  dbgm.GroupID,
		MemberID: dbgm.MemberID,
		Role:     dbgm.Role,
	}
}
