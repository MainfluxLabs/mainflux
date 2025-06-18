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

var _ things.GroupMembersRepository = (*membersRepository)(nil)

type membersRepository struct {
	db dbutil.Database
}

// NewGroupMembersRepository instantiates a PostgreSQL implementation of members repository.
func NewGroupMembersRepository(db dbutil.Database) things.GroupMembersRepository {
	return &membersRepository{
		db: db,
	}
}

func (mr membersRepository) Save(ctx context.Context, gms ...things.GroupMember) error {
	tx, err := mr.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	q := `INSERT INTO group_roles (member_id, group_id, role) VALUES (:member_id, :group_id, :role);`

	for _, g := range gms {
		dbgm := toDBGroupMembers(g)
		if _, err := mr.db.NamedExecContext(ctx, q, dbgm); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.ForeignKeyViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				case pgerrcode.UniqueViolation:
					return errors.Wrap(errors.ErrConflict, errors.New(pgErr.Detail))
				}
			}
			return errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (mr membersRepository) RetrieveRole(ctx context.Context, gp things.GroupMember) (string, error) {
	q := `SELECT role FROM group_roles WHERE member_id = $1 AND group_id = $2;`

	var role string
	if err := mr.db.QueryRowxContext(ctx, q, gp.MemberID, gp.GroupID).Scan(&role); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return "", errors.Wrap(errors.ErrNotFound, err)
		}

		return "", errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return role, nil
}

func (mr membersRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembersPage, error) {
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	q := fmt.Sprintf(`SELECT member_id, role FROM group_roles WHERE group_id = :group_id %s;`, olq)

	params := map[string]interface{}{
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	rows, err := mr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.GroupMembersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMember
	for rows.Next() {
		dbgp := dbGroupMembers{}
		if err := rows.StructScan(&dbgp); err != nil {
			return things.GroupMembersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupMembers(dbgp)
		items = append(items, gp)
	}

	cq := `SELECT COUNT(*) FROM group_roles WHERE group_id = :group_id;`

	total, err := dbutil.Total(ctx, mr.db, cq, params)
	if err != nil {
		return things.GroupMembersPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.GroupMembersPage{
		GroupMembers: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (mr membersRepository) RetrieveAll(ctx context.Context) ([]things.GroupMember, error) {
	q := `SELECT member_id, group_id, role FROM group_roles;`

	rows, err := mr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []things.GroupMember
	for rows.Next() {
		dbgp := dbGroupMembers{}
		if err := rows.StructScan(&dbgp); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		gp := toGroupMembers(dbgp)
		items = append(items, gp)
	}

	return items, nil
}

func (mr membersRepository) RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	var groupIDs []string

	q := `SELECT group_id FROM group_roles WHERE member_id = $1;`

	if err := mr.db.SelectContext(ctx, &groupIDs, q, memberID); err != nil {
		return nil, err
	}

	return groupIDs, nil
}

func (mr membersRepository) Remove(ctx context.Context, groupID string, memberIDs ...string) error {
	q := `DELETE FROM group_roles WHERE member_id = :member_id AND group_id = :group_id;`

	for _, memberID := range memberIDs {
		dbgp := dbGroupMembers{
			MemberID: memberID,
			GroupID:  groupID,
		}

		if _, err := mr.db.NamedExecContext(ctx, q, dbgp); err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (mr membersRepository) Update(ctx context.Context, gms ...things.GroupMember) error {
	q := `UPDATE group_roles SET role = :role WHERE member_id = :member_id AND group_id = :group_id;`

	for _, g := range gms {
		dbgm := toDBGroupMembers(g)
		row, err := mr.db.NamedExecContext(ctx, q, dbgm)
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

		cnt, err := row.RowsAffected()
		if err != nil {
			return errors.Wrap(errors.ErrUpdateEntity, err)
		}

		if cnt != 1 {
			return errors.Wrap(errors.ErrNotFound, err)
		}
	}

	return nil
}

type dbGroupMembers struct {
	MemberID string `db:"member_id"`
	GroupID  string `db:"group_id"`
	Role     string `db:"role"`
}

func toDBGroupMembers(gp things.GroupMember) dbGroupMembers {
	return dbGroupMembers{
		MemberID: gp.MemberID,
		GroupID:  gp.GroupID,
		Role:     gp.Role,
	}
}

func toGroupMembers(dbgp dbGroupMembers) things.GroupMember {
	return things.GroupMember{
		GroupID:  dbgp.GroupID,
		MemberID: dbgp.MemberID,
		Role:     dbgp.Role,
	}
}
