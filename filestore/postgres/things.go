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

var _ filestore.ThingsRepository = (*thingsRepository)(nil)

type thingsRepository struct {
	db dbutil.Database
}

// NewThingsRepository instantiates a PostgreSQL implementation of filestore
// repository.
func NewThingsRepository(db dbutil.Database) filestore.ThingsRepository {
	return &thingsRepository{
		db: db,
	}
}

func (tr thingsRepository) Save(ctx context.Context, thingID, groupID string, fi filestore.FileInfo) error {
	q := `INSERT INTO things_files (file_name, file_class, file_format, thing_id, time, metadata, group_id)
		VALUES (:file_name, :file_class, :file_format, :thing_id, :time, :metadata, :group_id)`

	dbFile, err := toDBThingFile(thingID, fi)
	if err != nil {
		return err
	}
	dbFile.GroupID = groupID

	if _, err := tr.db.NamedExecContext(ctx, q, dbFile); err != nil {
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

func (tr thingsRepository) Update(ctx context.Context, thingID string, fi filestore.FileInfo) error {
	q := `UPDATE things_files SET metadata = :metadata, time = :time
          WHERE thing_id = :thing_id AND file_name = :file_name AND file_class = :file_class AND file_format = :file_format`

	dbFile, err := toDBThingFile(thingID, fi)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := tr.db.NamedExecContext(ctx, q, dbFile)
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

func (tr thingsRepository) Retrieve(ctx context.Context, thingID string, fi filestore.FileInfo) (filestore.FileInfo, error) {
	q := `SELECT file_name, file_class, file_format, metadata FROM things_files
		WHERE thing_id = $1 AND file_class = $2 AND file_format = $3 AND file_name = $4`

	dbFile := dbFileInfo{}
	if err := tr.db.QueryRowxContext(ctx, q, thingID, fi.Class, fi.Format, fi.Name).StructScan(&dbFile); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return filestore.FileInfo{}, errors.Wrap(dbutil.ErrNotFound, err)
		}

		return filestore.FileInfo{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toFileInfo(dbFile)
}

func (tr thingsRepository) RetrieveByThing(ctx context.Context, thingID string, fi filestore.FileInfo, pm filestore.PageMetadata) (filestore.FileThingsPage, error) {
	thq := getThingQuery(thingID)
	nq, name := getFileNameQuery(fi.Name)
	cq := getClassQuery(fi.Class)
	fq := getFormatQuery(fi.Format)
	oq := getFileOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	var query []string
	if thq != "" {
		query = append(query, thq)
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
	q := fmt.Sprintf(`SELECT file_name, file_class, file_format, time, metadata FROM things_files %s ORDER BY %s %s %s`, whereClause, oq, dq, olq)

	params := map[string]any{
		"thing_id":    thingID,
		"file_name":   name,
		"file_class":  fi.Class,
		"file_format": fi.Format,
		"limit":       pm.Limit,
		"offset":      pm.Offset,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return filestore.FileThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var files []filestore.FileInfo
	for rows.Next() {
		var dbfi dbFileInfo
		if err := rows.StructScan(&dbfi); err != nil {
			return filestore.FileThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		file, err := toFileInfo(dbfi)
		if err != nil {
			return filestore.FileThingsPage{}, err
		}
		files = append(files, file)
	}

	tq := fmt.Sprintf(`SELECT COUNT(*) FROM things_files %s;`, whereClause)

	var total uint64
	rws, err := tr.db.NamedQueryContext(ctx, tq, params)
	if err != nil {
		return filestore.FileThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rws.Close()
	if rws.Next() {
		if err := rws.Scan(&total); err != nil {
			return filestore.FileThingsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
	}

	return filestore.FileThingsPage{
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

func (tr thingsRepository) Remove(ctx context.Context, thingID string, fi filestore.FileInfo) error {
	dbFile := dbThingFile{
		ThingID: thingID,
		dbFileInfo: dbFileInfo{
			FileClass:  fi.Class,
			FileFormat: fi.Format,
			FileName:   fi.Name,
		},
	}

	q := `DELETE FROM things_files WHERE thing_id = :thing_id AND file_class = :file_class AND file_format = :file_format AND file_name = :file_name`

	if _, err := tr.db.NamedExecContext(ctx, q, dbFile); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}
	return nil
}

func (tr thingsRepository) RemoveByThing(ctx context.Context, thingID string) error {
	dbFile := dbThingFile{ThingID: thingID}

	q := `DELETE FROM things_files WHERE thing_id = :thing_id`

	if _, err := tr.db.NamedExecContext(ctx, q, dbFile); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tr thingsRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	dbFile := dbThingFile{GroupID: groupID}

	q := `DELETE FROM things_files WHERE group_id = :group_id`

	if _, err := tr.db.NamedExecContext(ctx, q, dbFile); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (tr thingsRepository) RetrieveThingIDsByGroup(ctx context.Context, groupID string) ([]string, error) {
	var thingIDs []string

	q := `SELECT DISTINCT thing_id FROM things_files WHERE group_id = $1`

	if err := tr.db.SelectContext(ctx, &thingIDs, q, groupID); err != nil {
		return nil, err
	}

	return thingIDs, nil
}

type dbFileInfo struct {
	FileName   string  `db:"file_name"`
	FileClass  string  `db:"file_class"`
	FileFormat string  `db:"file_format"`
	Metadata   []byte  `db:"metadata"`
	Time       float64 `db:"time"`
}

type dbThingFile struct {
	dbFileInfo
	ThingID string `db:"thing_id"`
	GroupID string `db:"group_id"`
}

func toDBThingFile(thingID string, fi filestore.FileInfo) (dbThingFile, error) {
	meta := []byte("{}")
	if len(fi.Metadata) > 0 {
		b, err := json.Marshal(fi.Metadata)
		if err != nil {
			return dbThingFile{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		meta = b
	}

	return dbThingFile{
		dbFileInfo: dbFileInfo{
			FileName:   fi.Name,
			FileClass:  fi.Class,
			FileFormat: fi.Format,
			Metadata:   meta,
			Time:       fi.Time,
		},
		ThingID: thingID,
	}, nil
}

func toFileInfo(dbfi dbFileInfo) (filestore.FileInfo, error) {
	var metadata map[string]any
	if err := json.Unmarshal(dbfi.Metadata, &metadata); err != nil {
		return filestore.FileInfo{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	fileInfo := filestore.FileInfo{
		Name:     dbfi.FileName,
		Class:    dbfi.FileClass,
		Format:   dbfi.FileFormat,
		Metadata: metadata,
		Time:     dbfi.Time,
	}

	return fileInfo, nil
}
