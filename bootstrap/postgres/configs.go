// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/bootstrap"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	errSaveChannels    = errors.New("failed to insert channels to database")
	errSaveConnections = errors.New("failed to insert connections to database")
	errUpdateChannels  = errors.New("failed to update channels in bootstrap configuration database")
	errRemoveChannels  = errors.New("failed to remove channels from bootstrap configuration in database")
	errDisconnectThing = errors.New("failed to disconnect thing in bootstrap configuration in database")
)

const cleanupQuery = `DELETE FROM channels ch WHERE NOT EXISTS (
						 SELECT channel_id FROM connections c WHERE ch.mainflux_channel = c.channel_id);`

var _ bootstrap.ConfigRepository = (*configRepository)(nil)

type configRepository struct {
	db  *sqlx.DB
	log logger.Logger
}

// NewConfigRepository instantiates a PostgreSQL implementation of config
// repository.
func NewConfigRepository(db *sqlx.DB, log logger.Logger) bootstrap.ConfigRepository {
	return &configRepository{db: db, log: log}
}

func (cr configRepository) Save(cfg bootstrap.Config, chsConnIDs []string) (string, error) {
	q := `INSERT INTO configs (mainflux_thing, owner, name, client_cert, client_key, ca_cert, mainflux_key, external_id, external_key, content, state)
		  VALUES (:mainflux_thing, :owner, :name, :client_cert, :client_key, :ca_cert, :mainflux_key, :external_id, :external_key, :content, :state)`

	tx, err := cr.db.Beginx()
	if err != nil {
		return "", errors.Wrap(errors.ErrCreateEntity, err)
	}

	dbcfg := toDBConfig(cfg)

	if _, err := tx.NamedExec(q, dbcfg); err != nil {
		e := err
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == pgerrcode.UniqueViolation {
			e = errors.ErrConflict
		}

		cr.rollback("Failed to insert a Config", tx)
		return "", errors.Wrap(errors.ErrCreateEntity, e)
	}

	if err := insertChannels(cfg.Owner, cfg.Channels, tx); err != nil {
		cr.rollback("Failed to insert Channels", tx)
		return "", errors.Wrap(errSaveChannels, err)
	}

	if err := insertConnections(cfg, chsConnIDs, tx); err != nil {
		cr.rollback("Failed to insert connections", tx)
		return "", errors.Wrap(errSaveConnections, err)
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config save", tx)
		return "", err
	}

	return cfg.ThingID, nil
}

func (cr configRepository) RetrieveByID(owner, id string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, external_id, external_key, name, content, state
		  FROM configs
		  WHERE mainflux_thing = $1 AND owner = $2`

	dbcfg := dbConfig{
		ThingID: id,
		Owner:   owner,
	}

	if err := cr.db.QueryRowx(q, id, owner).StructScan(&dbcfg); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, errors.Wrap(errors.ErrNotFound, err)
		}

		return empty, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
		 INNER JOIN connections conn
		 ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
		 WHERE conn.config_id = :mainflux_thing AND conn.config_owner = :owner`

	rows, err := cr.db.NamedQuery(q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	chans := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		dbch.Owner = nullString(dbcfg.Owner)

		ch, err := toChannel(dbch)
		if err != nil {
			return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}
		chans = append(chans, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.Channels = chans

	return cfg, nil
}

func (cr configRepository) RetrieveAll(owner string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	search, params := cr.retrieveAll(owner, filter)
	n := len(params)

	q := `SELECT mainflux_thing, mainflux_key, external_id, external_key, name, content, state
	      FROM configs %s ORDER BY mainflux_thing LIMIT $%d OFFSET $%d`
	q = fmt.Sprintf(q, search, n+1, n+2)

	rows, err := cr.db.Query(q, append(params, limit, offset)...)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}
	defer rows.Close()

	var name, content sql.NullString
	configs := []bootstrap.Config{}

	for rows.Next() {
		c := bootstrap.Config{Owner: owner}
		if err := rows.Scan(&c.ThingID, &c.ThingKey, &c.ExternalID, &c.ExternalKey, &name, &content, &c.State); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved config due to %s", err))
			return bootstrap.ConfigsPage{}
		}

		c.Name = name.String
		c.Content = content.String
		configs = append(configs, c)
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM configs %s`, search)

	var total uint64
	if err := cr.db.QueryRow(q, params...).Scan(&total); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to count configs due to %s", err))
		return bootstrap.ConfigsPage{}
	}

	return bootstrap.ConfigsPage{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Configs: configs,
	}
}

func (cr configRepository) RetrieveByExternalID(externalID string) (bootstrap.Config, error) {
	q := `SELECT mainflux_thing, mainflux_key, external_key, owner, name, client_cert, client_key, ca_cert, content, state
		  FROM configs
		  WHERE external_id = $1`
	dbcfg := dbConfig{
		ExternalID: externalID,
	}

	if err := cr.db.QueryRowx(q, externalID).StructScan(&dbcfg); err != nil {
		empty := bootstrap.Config{}
		if err == sql.ErrNoRows {
			return empty, errors.Wrap(errors.ErrNotFound, err)
		}
		return empty, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	q = `SELECT mainflux_channel, name, metadata FROM channels ch
		 INNER JOIN connections conn
		 ON ch.mainflux_channel = conn.channel_id AND ch.owner = conn.config_owner
		 WHERE conn.config_id = :mainflux_thing AND conn.config_owner = :owner`

	rows, err := cr.db.NamedQuery(q, dbcfg)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve connected due to %s", err))
		return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}
	defer rows.Close()

	channels := []bootstrap.Channel{}
	for rows.Next() {
		dbch := dbChannel{}
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read connected thing due to %s", err))
			return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			cr.log.Error(fmt.Sprintf("Failed to deserialize channel due to %s", err))
			return bootstrap.Config{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		channels = append(channels, ch)
	}

	cfg := toConfig(dbcfg)
	cfg.Channels = channels

	return cfg, nil
}

func (cr configRepository) Update(cfg bootstrap.Config) error {
	q := `UPDATE configs SET name = $1, content = $2, external_id = $3, external_key = $4 WHERE mainflux_thing = $5 AND owner = $6`

	content := nullString(cfg.Content)
	name := nullString(cfg.Name)

	res, err := cr.db.Exec(q, name, content, cfg.ExternalID, cfg.ExternalKey, cfg.ThingID, cfg.Owner)
	if err != nil {
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

func (cr configRepository) UpdateCert(owner, thingID, clientCert, clientKey, caCert string) error {
	q := `UPDATE configs SET client_cert = $1, client_key = $2, ca_cert = $3 WHERE mainflux_thing = $4 AND owner = $5`

	res, err := cr.db.Exec(q, clientCert, clientKey, caCert, thingID, owner)
	if err != nil {
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

func (cr configRepository) UpdateConnections(owner, id string, channels []bootstrap.Channel, connections []string) error {
	tx, err := cr.db.Beginx()
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if err := insertChannels(owner, channels, tx); err != nil {
		cr.rollback("Failed to insert Channels during the update", tx)
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if err := updateConnections(owner, id, connections, tx); err != nil {
		if e, ok := err.(*pgconn.PgError); ok {
			if e.Code == pgerrcode.ForeignKeyViolation {
				return errors.ErrNotFound
			}
		}
		cr.rollback("Failed to update connections during the update", tx)
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	if err := tx.Commit(); err != nil {
		cr.rollback("Failed to commit Config update", tx)
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	return nil
}

func (cr configRepository) Remove(owner, id string) error {
	q := `DELETE FROM configs WHERE mainflux_thing = $1 AND owner = $2`
	if _, err := cr.db.Exec(q, id, owner); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	if _, err := cr.db.Exec(cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}

	return nil
}

func (cr configRepository) ChangeState(owner, id string, state bootstrap.State) error {
	q := `UPDATE configs SET state = $1 WHERE mainflux_thing = $2 AND owner = $3;`

	res, err := cr.db.Exec(q, state, id, owner)
	if err != nil {
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

func (cr configRepository) ListExisting(owner string, ids []string) ([]bootstrap.Channel, error) {
	var channels []bootstrap.Channel
	if len(ids) == 0 {
		return channels, nil
	}

	var chans pgtype.TextArray
	if err := chans.Set(ids); err != nil {
		return []bootstrap.Channel{}, err
	}

	q := "SELECT mainflux_channel, name, metadata FROM channels WHERE owner = $1 AND mainflux_channel = ANY ($2)"
	rows, err := cr.db.Queryx(q, owner, chans)
	if err != nil {
		return []bootstrap.Channel{}, errors.Wrap(errors.ErrRetrieveEntity, err)
	}

	for rows.Next() {
		var dbch dbChannel
		if err := rows.StructScan(&dbch); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved channels due to %s", err))
			return []bootstrap.Channel{}, errors.Wrap(errors.ErrRetrieveEntity, err)
		}

		ch, err := toChannel(dbch)
		if err != nil {
			cr.log.Error(fmt.Sprintf("Failed to deserialize channel due to %s", err))
			return []bootstrap.Channel{}, err
		}

		channels = append(channels, ch)
	}

	return channels, nil
}

func (cr configRepository) RemoveThing(id string) error {
	q := `DELETE FROM configs WHERE mainflux_thing = $1`
	_, err := cr.db.Exec(q, id)

	if _, err := cr.db.Exec(cleanupQuery); err != nil {
		cr.log.Warn("Failed to clean dangling channels after removal")
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

func (cr configRepository) UpdateChannel(c bootstrap.Channel) error {
	dbch, err := toDBChannel("", c)
	if err != nil {
		return errors.Wrap(errors.ErrUpdateEntity, err)
	}

	q := `UPDATE channels SET name = :name, metadata = :metadata WHERE mainflux_channel = :mainflux_channel`
	if _, err = cr.db.NamedExec(q, dbch); err != nil {
		return errors.Wrap(errUpdateChannels, err)
	}
	return nil
}

func (cr configRepository) RemoveChannel(id string) error {
	q := `DELETE FROM channels WHERE mainflux_channel = $1`
	if _, err := cr.db.Exec(q, id); err != nil {
		return errors.Wrap(errRemoveChannels, err)
	}
	return nil
}

func (cr configRepository) DisconnectThing(channelID, thingID string) error {
	q := `UPDATE configs SET state = $1 WHERE EXISTS (
		SELECT 1 FROM connections WHERE config_id = $2 AND channel_id = $3)`
	if _, err := cr.db.Exec(q, bootstrap.Inactive, thingID, channelID); err != nil {
		return errors.Wrap(errDisconnectThing, err)
	}
	return nil
}

func (cr configRepository) retrieveAll(owner string, filter bootstrap.Filter) (string, []interface{}) {
	template := `WHERE owner = $1 %s`
	params := []interface{}{owner}
	// One empty string so that strings Join works if only one filter is applied.
	queries := []string{""}
	// Since owner is the first param, start from 2.
	counter := 2
	for k, v := range filter.FullMatch {
		queries = append(queries, fmt.Sprintf("%s = $%d", k, counter))
		params = append(params, v)
		counter++
	}
	for k, v := range filter.PartialMatch {
		queries = append(queries, fmt.Sprintf("LOWER(%s) LIKE '%%' || $%d || '%%'", k, counter))
		params = append(params, v)
		counter++
	}

	f := strings.Join(queries, " AND ")

	return fmt.Sprintf(template, f), params
}

func (cr configRepository) rollback(content string, tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil {
		cr.log.Error(fmt.Sprintf("Failed to rollback due to %s", err))
	}
}

func insertChannels(owner string, channels []bootstrap.Channel, tx *sqlx.Tx) error {
	if len(channels) == 0 {
		return nil
	}

	var chans []dbChannel
	for _, ch := range channels {
		dbch, err := toDBChannel(owner, ch)
		if err != nil {
			return err
		}
		chans = append(chans, dbch)
	}

	q := `INSERT INTO channels (mainflux_channel, owner, name, metadata)
		  VALUES (:mainflux_channel, :owner, :name, :metadata)`
	if _, err := tx.NamedExec(q, chans); err != nil {
		e := err
		if pqErr, ok := err.(*pgconn.PgError); ok && pqErr.Code == pgerrcode.UniqueViolation {
			e = errors.ErrConflict
		}
		return e
	}

	return nil
}

func insertConnections(cfg bootstrap.Config, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner)
		  VALUES (:config_id, :channel_id, :config_owner, :channel_owner)`
	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:       cfg.ThingID,
			Channel:      conn,
			ConfigOwner:  cfg.Owner,
			ChannelOwner: cfg.Owner,
		}
		conns = append(conns, dbconn)
	}
	_, err := tx.NamedExec(q, conns)

	return err
}

func updateConnections(owner, id string, connections []string, tx *sqlx.Tx) error {
	if len(connections) == 0 {
		return nil
	}

	q := `DELETE FROM connections
		  WHERE config_id = $1 AND config_owner = $2 AND channel_owner = $2
		  AND channel_id NOT IN ($3)`

	var conn pgtype.TextArray
	if err := conn.Set(connections); err != nil {
		return err
	}

	res, err := tx.Exec(q, id, owner, conn)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	q = `INSERT INTO connections (config_id, channel_id, config_owner, channel_owner)
		 VALUES (:config_id, :channel_id, :config_owner, :channel_owner)`

	conns := []dbConnection{}
	for _, conn := range connections {
		dbconn := dbConnection{
			Config:       id,
			Channel:      conn,
			ConfigOwner:  owner,
			ChannelOwner: owner,
		}
		conns = append(conns, dbconn)
	}

	if _, err := tx.NamedExec(q, conns); err != nil {
		return err
	}

	if cnt == 0 {
		return nil
	}

	_, err = tx.Exec(cleanupQuery)

	return err
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

type dbConfig struct {
	ThingID     string          `db:"mainflux_thing"`
	ThingKey    string          `db:"mainflux_key"`
	Owner       string          `db:"owner"`
	Name        sql.NullString  `db:"name"`
	ClientCert  sql.NullString  `db:"client_cert"`
	ClientKey   sql.NullString  `db:"client_key"`
	CaCert      sql.NullString  `db:"ca_cert"`
	ExternalID  string          `db:"external_id"`
	ExternalKey string          `db:"external_key"`
	Content     sql.NullString  `db:"content"`
	State       bootstrap.State `db:"state"`
}

func toDBConfig(cfg bootstrap.Config) dbConfig {
	return dbConfig{
		ThingID:     cfg.ThingID,
		Owner:       cfg.Owner,
		Name:        nullString(cfg.Name),
		ClientCert:  nullString(cfg.ClientCert),
		ClientKey:   nullString(cfg.ClientKey),
		CaCert:      nullString(cfg.CACert),
		ThingKey:    cfg.ThingKey,
		ExternalID:  cfg.ExternalID,
		ExternalKey: cfg.ExternalKey,
		Content:     nullString(cfg.Content),
		State:       cfg.State,
	}
}

func toConfig(dbcfg dbConfig) bootstrap.Config {
	cfg := bootstrap.Config{
		ThingID:     dbcfg.ThingID,
		Owner:       dbcfg.Owner,
		ThingKey:    dbcfg.ThingKey,
		ExternalID:  dbcfg.ExternalID,
		ExternalKey: dbcfg.ExternalKey,
		State:       dbcfg.State,
	}

	if dbcfg.Name.Valid {
		cfg.Name = dbcfg.Name.String
	}

	if dbcfg.Content.Valid {
		cfg.Content = dbcfg.Content.String
	}

	if dbcfg.ClientCert.Valid {
		cfg.ClientCert = dbcfg.ClientCert.String
	}

	if dbcfg.ClientKey.Valid {
		cfg.ClientKey = dbcfg.ClientKey.String
	}

	if dbcfg.CaCert.Valid {
		cfg.CACert = dbcfg.CaCert.String
	}
	return cfg
}

type dbChannel struct {
	ID       string         `db:"mainflux_channel"`
	Name     sql.NullString `db:"name"`
	Owner    sql.NullString `db:"owner"`
	Metadata string         `db:"metadata"`
}

func toDBChannel(owner string, ch bootstrap.Channel) (dbChannel, error) {
	dbch := dbChannel{
		ID:    ch.ID,
		Name:  nullString(ch.Name),
		Owner: nullString(owner),
	}

	metadata, err := json.Marshal(ch.Metadata)
	if err != nil {
		return dbChannel{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	dbch.Metadata = string(metadata)
	return dbch, nil
}

func toChannel(dbch dbChannel) (bootstrap.Channel, error) {
	ch := bootstrap.Channel{
		ID: dbch.ID,
	}

	if dbch.Name.Valid {
		ch.Name = dbch.Name.String
	}

	if err := json.Unmarshal([]byte(dbch.Metadata), &ch.Metadata); err != nil {
		return bootstrap.Channel{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return ch, nil
}

type dbConnection struct {
	Config       string `db:"config_id"`
	Channel      string `db:"channel_id"`
	ConfigOwner  string `db:"config_owner"`
	ChannelOwner string `db:"channel_owner"`
}
