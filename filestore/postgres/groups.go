package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ filestore.GroupsRepository = (*groupsRepository)(nil)

type groupsRepository struct {
	db dbutil.Database
}

// NewGroupsRepository instantiates a PostgreSQL implementation of filestore
// repository.
func NewGroupsRepository(db dbutil.Database) filestore.GroupsRepository {
	return &groupsRepository{
		db: db,
	}
}

func (gr groupsRepository) Save(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	q := `INSERT INTO groups_files (file_name, file_class, file_format, group_id, time, metadata)
		VALUES (:file_name, :file_class, :file_format, :group_id, :time, :metadata)`

	dbFile, err := toDBFileInfoGroups(groupID, fi)
	if err != nil {
		return err
	}

	if _, err := gr.db.NamedExecContext(ctx, q, dbFile); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return errors.Wrap(dbutil.ErrConflict, err)
			}
		}

		return errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return nil
}

func (gr groupsRepository) Update(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	q := `UPDATE groups_files SET metadata = :metadata, time = :time
          WHERE group_id = :group_id AND file_name = :file_name AND file_class = :file_class AND file_format = :file_format`

	dbFile, err := toDBFileInfoGroups(groupID, fi)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := gr.db.NamedExecContext(ctx, q, dbFile)
	if errdb != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, err)
			}
		}

		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return dbutil.ErrNotFound
	}

	return nil
}

func (gr groupsRepository) Retrieve(ctx context.Context, groupID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	q := `SELECT file_name, file_class, file_format, metadata FROM groups_files
		WHERE group_id = $1 AND file_class = $2 AND file_format = $3 AND file_name = $4`

	dbFile := dbFileInfo{}
	if err := gr.db.QueryRowxContext(ctx, q, groupID, fi.Class, fi.Format, fi.Name).StructScan(&dbFile); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return filestore.FileInfo{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		return filestore.FileInfo{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toFileInfo(dbFile)
}

func (gr groupsRepository) RetrieveByGroup(ctx context.Context, groupID string, fi filestore.FileInfo, pm filestore.PageMetadata) (filestore.FileGroupsPage, error) {
	grq := getGroupQuery(groupID)
	nq, name := getFileNameQuery(fi.Name)
	cq := getClassQuery(fi.Class)
	fq := getFormatQuery(fi.Format)
	oq := getFileOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := "LIMIT :limit OFFSET :offset"
	if pm.Limit == 0 {
		olq = ""
	}

	var query []string
	if grq != "" {
		query = append(query, grq)
	}
	if nq != "" {
		query = append(query, nq)
	}
	if cq != "" {
		query = append(query, cq)
	}
	if fq != "" {
		query = append(query, fq)
	}
	var whereClause string
	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}
	q := fmt.Sprintf(`SELECT file_name, file_class, file_format, time, metadata FROM groups_files %s ORDER BY %s %s %s`, whereClause, oq, dq, olq)

	params := map[string]any{
		"group_id":    groupID,
		"file_name":   name,
		"file_class":  fi.Class,
		"file_format": fi.Format,
		"limit":       pm.Limit,
		"offset":      pm.Offset,
	}

	rows, err := gr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return filestore.FileGroupsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var files []filestore.FileInfo
	for rows.Next() {
		var dbfi dbFileInfo
		if err := rows.StructScan(&dbfi); err != nil {
			return filestore.FileGroupsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		file, err := toFileInfo(dbfi)
		if err != nil {
			return filestore.FileGroupsPage{}, err
		}
		files = append(files, file)
	}

	tq := fmt.Sprintf(`SELECT COUNT(*) FROM groups_files %s;`, whereClause)

	var total uint64
	rws, err := gr.db.NamedQueryContext(ctx, tq, params)
	if err != nil {
		return filestore.FileGroupsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rws.Close()
	if rws.Next() {
		if err := rws.Scan(&total); err != nil {
			return filestore.FileGroupsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
	}

	return filestore.FileGroupsPage{
		PageMetadata: filestore.PageMetadata{
			Total:  total,
			Order:  pm.Order,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Dir:    pm.Dir,
		},
		Files: files,
	}, nil
}

func (gr groupsRepository) Remove(ctx context.Context, groupID string, fi filestore.FileInfo) error {
	dbFile := dbFileInfoGroups{
		GroupID: groupID,
		dbFileInfo: dbFileInfo{
			FileClass:  fi.Class,
			FileFormat: fi.Format,
			FileName:   fi.Name,
		},
	}

	q := `DELETE FROM groups_files WHERE group_id = :group_id AND file_class = :file_class AND file_format = :file_format AND file_name = :file_name`

	if _, err := gr.db.NamedExecContext(ctx, q, dbFile); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	return nil
}

func (gr groupsRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	params := map[string]any{"group_id": groupID}

	q := `DELETE FROM groups_files WHERE group_id = :group_id`

	if _, err := gr.db.NamedExecContext(ctx, q, params); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

type dbFileInfoGroups struct {
	dbFileInfo
	GroupID string `db:"group_id"`
}

func toDBFileInfoGroups(groupID string, fi filestore.FileInfo) (dbFileInfoGroups, error) {
	meta := []byte("{}")
	if len(fi.Metadata) > 0 {
		b, err := json.Marshal(fi.Metadata)
		if err != nil {
			return dbFileInfoGroups{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		meta = b
	}

	return dbFileInfoGroups{
		dbFileInfo: dbFileInfo{
			FileName:   fi.Name,
			FileClass:  fi.Class,
			FileFormat: fi.Format,
			Metadata:   meta,
			Time:       fi.Time,
		},
		GroupID: groupID,
	}, nil
}
