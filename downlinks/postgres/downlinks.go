package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _ downlinks.DownlinkRepository = (*downlinkRepository)(nil)

type downlinkRepository struct {
	db dbutil.Database
}

// NewDownlinkRepository instantiates a PostgreSQL implementation of downlink repository.
func NewDownlinkRepository(db dbutil.Database) downlinks.DownlinkRepository {
	return &downlinkRepository{
		db: db,
	}
}

func (dr downlinkRepository) Save(ctx context.Context, dls ...downlinks.Downlink) ([]downlinks.Downlink, error) {
	tx, err := dr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	q := `INSERT INTO downlinks (id, group_id, thing_id, name, url, method, payload, headers, 
          scheduler, time_filter, metadata) 
          VALUES (:id, :group_id, :thing_id, :name, :url, :method, :payload, :headers, 
          :scheduler, :time_filter, :metadata);`

	for _, downlink := range dls {
		dbDl, err := toDBDownlink(downlink)
		if err != nil {
			return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbDl); err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []downlinks.Downlink{}, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return dls, nil
}

func (dr downlinkRepository) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	if _, err := uuid.FromString(thingID); err != nil {
		return downlinks.DownlinksPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, downlinks.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	nq, name := dbutil.GetNameQuery(pm.Name)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	filters := []string{"thing_id = :thing_id"}
	if nq != "" {
		filters = append(filters, nq)
	}

	whereClause := dbutil.BuildWhereClause(filters...)
	query := fmt.Sprintf(`SELECT id, thing_id, group_id, name, url, method, payload, headers, 
    	  scheduler, time_filter, metadata
          FROM downlinks %s
          ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM downlinks %s`, whereClause)

	params := map[string]any{
		"name":     name,
		"thing_id": thingID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return dr.retrieve(ctx, query, cquery, params)
}

func (dr downlinkRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (downlinks.DownlinksPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return downlinks.DownlinksPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order, downlinks.AllowedOrders)
	dq := dbutil.GetDirQuery(pm.Dir)
	nq, name := dbutil.GetNameQuery(pm.Name)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	filters := []string{"group_id = :group_id"}
	if nq != "" {
		filters = append(filters, nq)
	}

	whereClause := dbutil.BuildWhereClause(filters...)
	query := fmt.Sprintf(`SELECT id, thing_id, group_id, name, url, method, payload, headers, 
    	  scheduler, time_filter, metadata
          FROM downlinks %s
          ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM downlinks %s`, whereClause)

	params := map[string]any{
		"name":     name,
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
	}

	return dr.retrieve(ctx, query, cquery, params)
}

func (dr downlinkRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (downlinks.DownlinksPage, error) {
	rows, err := dr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return downlinks.DownlinksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []downlinks.Downlink
	for rows.Next() {
		dbDl := dbDownlink{}
		if err := rows.StructScan(&dbDl); err != nil {
			return downlinks.DownlinksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		downlink, err := toDownlink(dbDl)
		if err != nil {
			return downlinks.DownlinksPage{}, err
		}

		items = append(items, downlink)
	}

	total, err := dbutil.Total(ctx, dr.db, cquery, params)
	if err != nil {
		return downlinks.DownlinksPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := downlinks.DownlinksPage{
		Downlinks: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: params["offset"].(uint64),
			Limit:  params["limit"].(uint64),
		},
	}

	return page, nil
}

func (dr downlinkRepository) RetrieveAll(ctx context.Context) ([]downlinks.Downlink, error) {
	q := `SELECT id, group_id, thing_id, name, url, method, payload, headers, 
    	  scheduler, time_filter, metadata 
          FROM downlinks`

	var items []dbDownlink
	err := dr.db.SelectContext(ctx, &items, q)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var dws []downlinks.Downlink
	for _, i := range items {
		dw, err := toDownlink(i)
		if err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		dws = append(dws, dw)
	}

	return dws, nil
}

func (dr downlinkRepository) RetrieveByID(ctx context.Context, id string) (downlinks.Downlink, error) {
	q := `SELECT group_id, thing_id, name, url, method, payload, headers, 
    	  scheduler, time_filter, metadata
          FROM downlinks 
          WHERE id = $1;`
	dbDl := dbDownlink{ID: id}
	if err := dr.db.QueryRowxContext(ctx, q, id).StructScan(&dbDl); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return downlinks.Downlink{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return downlinks.Downlink{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toDownlink(dbDl)
}

func (dr downlinkRepository) Update(ctx context.Context, w downlinks.Downlink) error {
	q := `UPDATE downlinks SET name = :name, url = :url, method = :method, payload = :payload, headers = :headers,
          scheduler = :scheduler, time_filter = :time_filter, metadata = :metadata
          WHERE id = :id;`

	dbDl, err := toDBDownlink(w)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := dr.db.NamedExecContext(ctx, q, dbDl)
	if errdb != nil {
		pgErr, ok := errdb.(*pgconn.PgError)
		if ok {
			switch pgErr.Code {
			case pgerrcode.InvalidTextRepresentation:
				return errors.Wrap(dbutil.ErrMalformedEntity, errdb)
			case pgerrcode.StringDataRightTruncationDataException:
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

func (dr downlinkRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbDl := dbDownlink{ID: id}
		q := `DELETE FROM downlinks WHERE id = :id;`

		if _, err := dr.db.NamedExecContext(ctx, q, dbDl); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (dr downlinkRepository) RemoveByThing(ctx context.Context, thingID string) error {
	dbDl := dbDownlink{ThingID: thingID}
	q := `DELETE FROM downlinks WHERE thing_id = :thing_id;`

	if _, err := dr.db.NamedExecContext(ctx, q, dbDl); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (dr downlinkRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	dbDl := dbDownlink{GroupID: groupID}
	q := `DELETE FROM downlinks WHERE group_id = :group_id;`

	if _, err := dr.db.NamedExecContext(ctx, q, dbDl); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

type dbDownlink struct {
	ID         string `db:"id"`
	GroupID    string `db:"group_id"`
	ThingID    string `db:"thing_id"`
	Name       string `db:"name"`
	Url        string `db:"url"`
	Method     string `db:"method"`
	Payload    []byte `db:"payload"`
	Headers    []byte `db:"headers"`
	Scheduler  []byte `db:"scheduler"`
	TimeFilter []byte `db:"time_filter"`
	Metadata   []byte `db:"metadata"`
}

func toDBDownlink(dl downlinks.Downlink) (dbDownlink, error) {
	headers := []byte("{}")
	if len(dl.Headers) > 0 {
		b, err := json.Marshal(dl.Headers)
		if err != nil {
			return dbDownlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		headers = b
	}

	metadata := []byte("{}")
	if len(dl.Metadata) > 0 {
		b, err := json.Marshal(dl.Metadata)
		if err != nil {
			return dbDownlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
		}
		metadata = b
	}

	scheduler, err := json.Marshal(dl.Scheduler)
	if err != nil {
		return dbDownlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	timeFilter, err := json.Marshal(dl.TimeFilter)
	if err != nil {
		return dbDownlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return dbDownlink{
		ID:         dl.ID,
		GroupID:    dl.GroupID,
		ThingID:    dl.ThingID,
		Name:       dl.Name,
		Url:        dl.Url,
		Method:     dl.Method,
		Payload:    dl.Payload,
		Headers:    headers,
		Scheduler:  scheduler,
		TimeFilter: timeFilter,
		Metadata:   metadata,
	}, nil
}

func toDownlink(dbD dbDownlink) (downlinks.Downlink, error) {
	var headers map[string]string
	if err := json.Unmarshal(dbD.Headers, &headers); err != nil {
		return downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(dbD.Metadata, &metadata); err != nil {
		return downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var scheduler cron.Scheduler
	if err := json.Unmarshal(dbD.Scheduler, &scheduler); err != nil {
		return downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var timeFilter downlinks.TimeFilter
	if err := json.Unmarshal(dbD.TimeFilter, &timeFilter); err != nil {
		return downlinks.Downlink{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return downlinks.Downlink{
		ID:         dbD.ID,
		GroupID:    dbD.GroupID,
		ThingID:    dbD.ThingID,
		Name:       dbD.Name,
		Url:        dbD.Url,
		Method:     dbD.Method,
		Payload:    dbD.Payload,
		Headers:    headers,
		Scheduler:  scheduler,
		TimeFilter: timeFilter,
		Metadata:   metadata,
	}, nil
}
