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

var _ things.ChannelRepository = (*channelRepository)(nil)

type channelRepository struct {
	db Database
}

type dbConnection struct {
	Channel string `db:"channel"`
	Thing   string `db:"thing"`
	GroupID string `db:"group_id"`
}

// NewChannelRepository instantiates a PostgreSQL implementation of channel
// repository.
func NewChannelRepository(db Database) things.ChannelRepository {
	return &channelRepository{
		db: db,
	}
}

func (cr channelRepository) Save(ctx context.Context, channels ...things.Channel) ([]things.Channel, error) {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCreateEntity, err)
	}

	q := `INSERT INTO channels (id, group_id, name, metadata, config)
		  VALUES (:id, :group_id, :name, :metadata, :config);`

	for _, channel := range channels {
		dbch := toDBChannel(channel)

		_, err = tx.NamedExecContext(ctx, q, dbch)
		if err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return []things.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
				case pgerrcode.UniqueViolation:
					return []things.Channel{}, errors.Wrap(errors.ErrConflict, err)
				case pgerrcode.StringDataRightTruncationDataException:
					return []things.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
				}
			}
			return []things.Channel{}, errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Channel{}, errors.Wrap(errors.ErrCreateEntity, err)
	}

	return channels, nil
}

func (cr channelRepository) Update(ctx context.Context, channel things.Channel) error {
	q := `UPDATE channels SET name = :name, metadata = :metadata, config = :config WHERE id = :id;`

	dbch := toDBChannel(channel)

	res, err := cr.db.NamedExecContext(ctx, q, dbch)
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

func (cr channelRepository) RetrieveByID(ctx context.Context, id string) (things.Channel, error) {
	q := `SELECT group_id, name, metadata, config FROM channels WHERE id = $1;`

	dbch := dbChannel{
		ID: id,
	}

	if err := cr.db.QueryRowxContext(ctx, q, id).StructScan(&dbch); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		//  If there is no result or ID is in an invalid format, return ErrNotFound.
		if err == sql.ErrNoRows || ok && pgerrcode.InvalidTextRepresentation == pgErr.Code {
			return things.Channel{}, errors.ErrNotFound
		}
		return things.Channel{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return toChannel(dbch), nil
}

func (cr channelRepository) RetrieveAll(ctx context.Context) ([]things.Channel, error) {
	chPage, err := cr.retrieve(ctx, []string{}, true, things.PageMetadata{})
	if err != nil {
		return []things.Channel{}, err
	}

	return chPage.Channels, nil
}

func (cr channelRepository) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ChannelsPage, error) {
	return cr.retrieve(ctx, []string{}, false, pm)
}

func (cr channelRepository) RetrieveByThing(ctx context.Context, thID string) (things.Channel, error) {
	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(thID); err != nil {
		return things.Channel{}, errors.Wrap(errors.ErrNotFound, err)
	}

	var q string
	q = fmt.Sprintf(`SELECT id, group_id, name, metadata, config FROM channels ch
		        INNER JOIN connections conn
		        ON ch.id = conn.channel_id
		        WHERE conn.thing_id = :thing;`)

	params := map[string]interface{}{
		"thing": thID,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Channel{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	var item things.Channel
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return things.Channel{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		item = toChannel(dbch)
	}

	return item, nil
}

func (cr channelRepository) Remove(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		dbch := dbChannel{
			ID: id,
		}
		q := `DELETE FROM channels WHERE id = :id`
		_, err := cr.db.NamedExecContext(ctx, q, dbch)
		if err != nil {
			return errors.Wrap(errors.ErrRemoveEntity, err)
		}
	}

	return nil
}

func (cr channelRepository) Connect(ctx context.Context, chID string, thIDs []string) error {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	q := `INSERT INTO connections (channel_id, thing_id) VALUES (:channel, :thing);`

	for _, thID := range thIDs {
		dbco := dbConnection{
			Channel: chID,
			Thing:   thID,
		}

		_, err := tx.NamedExecContext(ctx, q, dbco)
		if err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.ForeignKeyViolation:
					return errors.ErrNotFound
				case pgerrcode.UniqueViolation:
					return errors.ErrConflict
				}
			}

			return errors.Wrap(things.ErrConnect, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	return nil
}

func (cr channelRepository) Disconnect(ctx context.Context, chID string, thIDs []string) error {
	tx, err := cr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	q := `DELETE FROM connections
	      WHERE channel_id = :channel AND thing_id = :thing`

	for _, thID := range thIDs {
		dbco := dbConnection{
			Channel: chID,
			Thing:   thID,
		}

		res, err := tx.NamedExecContext(ctx, q, dbco)
		if err != nil {
			tx.Rollback()
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.ForeignKeyViolation:
					return errors.ErrNotFound
				case pgerrcode.UniqueViolation:
					return errors.ErrConflict
				}
			}
			return errors.Wrap(things.ErrDisconnect, err)
		}

		cnt, err := res.RowsAffected()
		if err != nil {
			return errors.Wrap(things.ErrDisconnect, err)
		}

		if cnt == 0 {
			return errors.ErrNotFound
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}

	return nil
}

func (cr channelRepository) RetrieveConnByThingKey(ctx context.Context, thingKey string) (things.Connection, error) {
	var thingID string
	q := `SELECT id FROM things WHERE key = $1`
	if err := cr.db.QueryRowxContext(ctx, q, thingKey).Scan(&thingID); err != nil {
		return things.Connection{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	q = fmt.Sprintf(`SELECT thing_id, channel_id FROM connections
		               WHERE thing_id = :thing;`)

	params := map[string]interface{}{
		"thing": thingID,
	}

	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Connection{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	dbch := dbConn{}
	for rows.Next() {
		if err := rows.StructScan(&dbch); err != nil {
			return things.Connection{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
	}

	if dbch.ChannelID == "" {
		return things.Connection{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	return things.Connection{ThingID: thingID, ChannelID: dbch.ChannelID}, nil
}

func (cr channelRepository) RetrieveAllConnections(ctx context.Context) ([]things.Connection, error) {
	q := `SELECT channel_id, thing_id FROM connections;`

	rows, err := cr.db.NamedQueryContext(ctx, q, map[string]interface{}{})
	if err != nil {
		return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	connections := make([]things.Connection, 0)
	for rows.Next() {
		dbco := dbConn{}
		if err := rows.StructScan(&dbco); err != nil {
			return nil, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		connections = append(connections, toConnection(dbco))
	}

	return connections, nil
}

func (cr channelRepository) RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm things.PageMetadata) (things.ChannelsPage, error) {
	if len(groupIDs) == 0 {
		return things.ChannelsPage{}, nil
	}

	chPage, err := cr.retrieve(ctx, groupIDs, false, pm)
	if err != nil {
		return things.ChannelsPage{}, err
	}

	return chPage, nil
}

func (cr channelRepository) retrieve(ctx context.Context, groupIDs []string, allRows bool, pm things.PageMetadata) (things.ChannelsPage, error) {
	idsq := getGroupIDsQuery(groupIDs)
	nq, name := dbutil.GetNameQuery(pm.Name)
	oq := dbutil.GetOrderQuery(pm.Order)
	dq := dbutil.GetDirQuery(pm.Dir)
	olq := dbutil.GetOffsetLimitQuery(pm.Limit)
	meta, mq, err := dbutil.GetMetadataQuery("", pm.Metadata)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
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

	q := fmt.Sprintf(`SELECT id, group_id, name, metadata, config FROM channels %s ORDER BY %s %s %s;`, whereClause, oq, dq, olq)

	if allRows {
		q = "SELECT id, group_id, name, metadata, config FROM channels"
	}

	params := map[string]interface{}{
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": meta,
	}
	rows, err := cr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	items := []things.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			return things.ChannelsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		ch := toChannel(dbch)

		items = append(items, ch)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM channels %s;`, whereClause)

	total, err := total(ctx, cr.db, cq, params)
	if err != nil {
		return things.ChannelsPage{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	page := things.ChannelsPage{
		Channels: items,
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

type dbChannel struct {
	ID       string  `db:"id"`
	GroupID  string  `db:"group_id"`
	Name     string  `db:"name"`
	Config   dbJSONB `db:"config"`
	Metadata dbJSONB `db:"metadata"`
}

func toDBChannel(ch things.Channel) dbChannel {
	return dbChannel{
		ID:       ch.ID,
		GroupID:  ch.GroupID,
		Name:     ch.Name,
		Config:   ch.Config,
		Metadata: ch.Metadata,
	}
}

func toChannel(ch dbChannel) things.Channel {
	return things.Channel{
		ID:       ch.ID,
		GroupID:  ch.GroupID,
		Name:     ch.Name,
		Config:   ch.Config,
		Metadata: ch.Metadata,
	}
}

type dbConn struct {
	ChannelID string `db:"channel_id"`
	ThingID   string `db:"thing_id"`
}

func toConnection(co dbConn) things.Connection {
	return things.Connection{
		ChannelID: co.ChannelID,
		ThingID:   co.ThingID,
	}
}

func getConnOrderQuery(order string, level string) string {
	switch order {
	case "name":
		return level + ".name"
	default:
		return level + ".id"
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
