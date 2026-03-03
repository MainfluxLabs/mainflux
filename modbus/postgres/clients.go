package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type clientRepository struct {
	db dbutil.Database
}

// NewClientRepository instantiates a PostgreSQL implementation of client repository.
func NewClientRepository(db dbutil.Database) modbus.ClientRepository {
	return &clientRepository{
		db: db,
	}
}

type dbClient struct {
	ID           string `db:"id"`
	GroupID      string `db:"group_id"`
	ThingID      string `db:"thing_id"`
	Name         string `db:"name"`
	IPAddress    string `db:"ip_address"`
	Port         string `db:"port"`
	SlaveID      uint8  `db:"slave_id"`
	FunctionCode string `db:"function_code"`
	Scheduler    []byte `db:"scheduler"`
	DataFields   []byte `db:"data_fields"`
	Metadata     []byte `db:"metadata"`
}

func (cr clientRepository) Save(ctx context.Context, cls ...modbus.Client) ([]modbus.Client, error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(dbutil.ErrCreateEntity, err)
	}

	q := `INSERT INTO clients (id, group_id, thing_id, name, ip_address, port, slave_id, function_code,
			  scheduler, data_fields, metadata)
			  VALUES (:id, :group_id, :thing_id, :name, :ip_address, :port, :slave_id, :function_code, 
			  :scheduler, :data_fields, :metadata)`

	for _, c := range cls {
		dbCl, err := toDBClient(c)
		if err != nil {
			return nil, errors.Wrap(dbutil.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbCl); err != nil {
			_ = tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return nil, errors.Wrap(dbutil.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return nil, errors.Wrap(dbutil.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationWarning:
					return nil, errors.Wrap(dbutil.ErrMalformedEntity, err)
				}
			}

			return nil, errors.Wrap(dbutil.ErrCreateEntity, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(dbutil.ErrCreateEntity, err)
	}
	return cls, nil
}

func (cr clientRepository) RetrieveAll(ctx context.Context) ([]modbus.Client, error) {
	query := `SELECT id, group_id, thing_id, name, ip_address, port, slave_id, function_code,
			  scheduler, data_fields, metadata 
			  FROM clients`

	var dbCls []dbClient
	if err := cr.db.SelectContext(ctx, &dbCls, query); err != nil {
		return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	var cls []modbus.Client
	for _, dbCl := range dbCls {
		cl, err := toClient(dbCl)
		if err != nil {
			return nil, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}

		cls = append(cls, cl)
	}

	return cls, nil
}

func (cr clientRepository) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	if _, err := uuid.FromString(thingID); err != nil {
		return modbus.ClientsPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	nq, name := dbutil.GetNameQuery(pm.Name)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	filters := []string{"thing_id = :thing_id"}
	if nq != "" {
		filters = append(filters, nq)
	}

	whereClause := dbutil.BuildWhereClause(filters...)
	query := fmt.Sprintf(`SELECT id, group_id, thing_id, name, ip_address, port, slave_id, function_code, 
          scheduler, data_fields, metadata
          FROM clients %s
          ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM clients %s`, whereClause)

	params := map[string]any{
		"name":     name,
		"thing_id": thingID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"order":    pm.Order,
		"dir":      pm.Dir,
	}

	return cr.retrieve(ctx, query, cquery, params)
}

func (cr clientRepository) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (modbus.ClientsPage, error) {
	if _, err := uuid.FromString(groupID); err != nil {
		return modbus.ClientsPage{}, errors.Wrap(dbutil.ErrNotFound, err)
	}

	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	nq, name := dbutil.GetNameQuery(pm.Name)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)

	filters := []string{"group_id = :group_id"}
	if nq != "" {
		filters = append(filters, nq)
	}

	whereClause := dbutil.BuildWhereClause(filters...)
	query := fmt.Sprintf(`SELECT id, group_id, thing_id, name, ip_address, port, slave_id, function_code, 
          scheduler, data_fields, metadata
          FROM clients %s
          ORDER BY %s %s %s`, whereClause, oq, dq, olq)
	cquery := fmt.Sprintf(`SELECT COUNT(*) FROM clients %s`, whereClause)

	params := map[string]any{
		"name":     name,
		"group_id": groupID,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"order":    pm.Order,
		"dir":      pm.Dir,
	}

	return cr.retrieve(ctx, query, cquery, params)
}

func (cr clientRepository) RetrieveByID(ctx context.Context, id string) (modbus.Client, error) {
	q := `SELECT id, group_id, thing_id, name, ip_address, port, slave_id, function_code, 
          scheduler, data_fields, metadata
          FROM clients 
          WHERE id = $1;`
	dbCl := dbClient{ID: id}
	if err := cr.db.QueryRowxContext(ctx, q, id).StructScan(&dbCl); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return modbus.Client{}, errors.Wrap(dbutil.ErrNotFound, err)
		}
		return modbus.Client{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	return toClient(dbCl)
}

func (cr clientRepository) retrieve(ctx context.Context, query, cquery string, params map[string]any) (modbus.ClientsPage, error) {
	rows, err := cr.db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return modbus.ClientsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var items []modbus.Client
	for rows.Next() {
		dbCl := dbClient{}
		if err := rows.StructScan(&dbCl); err != nil {
			return modbus.ClientsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
		}
		client, err := toClient(dbCl)
		if err != nil {
			return modbus.ClientsPage{}, err
		}

		items = append(items, client)
	}

	total, err := dbutil.Total(ctx, cr.db, cquery, params)
	if err != nil {
		return modbus.ClientsPage{}, errors.Wrap(dbutil.ErrRetrieveEntity, err)
	}

	page := modbus.ClientsPage{
		Clients: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  total,
			Offset: params["offset"].(uint64),
			Limit:  params["limit"].(uint64),
			Order:  params["order"].(string),
			Dir:    params["dir"].(string),
			Name:   params["name"].(string),
		},
	}

	return page, nil
}

func (cr clientRepository) Update(ctx context.Context, c modbus.Client) error {
	q := `UPDATE clients SET name = :name, ip_address = :ip_address, port = :port, slave_id = :slave_id, function_code = :function_code,
          scheduler = :scheduler, data_fields = :data_fields, metadata = :metadata
          WHERE id = :id;`

	dbCl, err := toDBClient(c)
	if err != nil {
		return errors.Wrap(dbutil.ErrUpdateEntity, err)
	}

	res, errdb := cr.db.NamedExecContext(ctx, q, dbCl)
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

func (cr clientRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbCl := dbClient{ID: id}
		q := `DELETE FROM clients WHERE id = :id;`

		if _, err := cr.db.NamedExecContext(ctx, q, dbCl); err != nil {
			return errors.Wrap(dbutil.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (cr clientRepository) RemoveByThing(ctx context.Context, thingID string) error {
	dbCl := dbClient{ThingID: thingID}
	q := `DELETE FROM clients WHERE thing_id = :thing_id;`

	if _, err := cr.db.NamedExecContext(ctx, q, dbCl); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func (cr clientRepository) RemoveByGroup(ctx context.Context, groupID string) error {
	dbCl := dbClient{GroupID: groupID}
	q := `DELETE FROM clients WHERE group_id = :group_id;`

	if _, err := cr.db.NamedExecContext(ctx, q, dbCl); err != nil {
		return errors.Wrap(dbutil.ErrRemoveEntity, err)
	}

	return nil
}

func toDBClient(c modbus.Client) (dbClient, error) {
	metadata, err := json.Marshal(c.Metadata)
	if err != nil {
		return dbClient{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	scheduler, err := json.Marshal(c.Scheduler)
	if err != nil {
		return dbClient{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	dataFields, err := json.Marshal(c.DataFields)
	if err != nil {
		return dbClient{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return dbClient{
		ID:           c.ID,
		GroupID:      c.GroupID,
		ThingID:      c.ThingID,
		Name:         c.Name,
		IPAddress:    c.IPAddress,
		Port:         c.Port,
		SlaveID:      c.SlaveID,
		FunctionCode: c.FunctionCode,
		Scheduler:    scheduler,
		DataFields:   dataFields,
		Metadata:     metadata,
	}, nil
}

func toClient(dbC dbClient) (modbus.Client, error) {
	var metadata map[string]any
	if err := json.Unmarshal(dbC.Metadata, &metadata); err != nil {
		return modbus.Client{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var scheduler cron.Scheduler
	if err := json.Unmarshal(dbC.Scheduler, &scheduler); err != nil {
		return modbus.Client{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	var dataFields []modbus.DataField
	if err := json.Unmarshal(dbC.DataFields, &dataFields); err != nil {
		return modbus.Client{}, errors.Wrap(dbutil.ErrMalformedEntity, err)
	}

	return modbus.Client{
		ID:           dbC.ID,
		GroupID:      dbC.GroupID,
		ThingID:      dbC.ThingID,
		Name:         dbC.Name,
		IPAddress:    dbC.IPAddress,
		Port:         dbC.Port,
		SlaveID:      dbC.SlaveID,
		FunctionCode: dbC.FunctionCode,
		Scheduler:    scheduler,
		DataFields:   dataFields,
		Metadata:     metadata,
	}, nil
}
