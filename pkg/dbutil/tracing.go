package dbutil

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
)

var _ Database = (*database)(nil)

type database struct {
	db *sqlx.DB
}

// Database provides a database interface
type Database interface {
	NamedExecContext(context.Context, string, any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowxContext(context.Context, string, ...any) *sqlx.Row
	NamedQueryContext(context.Context, string, any) (*sqlx.Rows, error)
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	GetContext(context.Context, any, string, ...any) error
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

// NewDatabase creates a ThingDatabase instance
func NewDatabase(db *sqlx.DB) Database {
	return &database{
		db: db,
	}
}

func (dm database) NamedExecContext(ctx context.Context, query string, args any) (sql.Result, error) {
	addSpanTags(ctx, query)
	return dm.db.NamedExecContext(ctx, query, args)
}

func (dm database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	addSpanTags(ctx, query)
	return dm.db.ExecContext(ctx, query, args...)
}

func (dm database) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	addSpanTags(ctx, query)
	return dm.db.QueryRowxContext(ctx, query, args...)
}

func (dm database) NamedQueryContext(ctx context.Context, query string, args any) (*sqlx.Rows, error) {
	addSpanTags(ctx, query)
	return dm.db.NamedQueryContext(ctx, query, args)
}

func (dm database) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	addSpanTags(ctx, query)
	return dm.db.SelectContext(ctx, dest, query, args...)
}

func (dm database) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	addSpanTags(ctx, query)
	return dm.db.GetContext(ctx, dest, query, args...)
}

func (dm database) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag("span.kind", "client")
		span.SetTag("peer.service", "postgres")
		span.SetTag("db.type", "sql")
	}
	return dm.db.BeginTxx(ctx, opts)
}

func addSpanTags(ctx context.Context, query string) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag("sql.statement", query)
		span.SetTag("span.kind", "client")
		span.SetTag("peer.service", "postgres")
		span.SetTag("db.type", "sql")
	}
}

func CreateSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
