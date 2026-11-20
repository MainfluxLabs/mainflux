package dbutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/jmoiron/sqlx"
)

// maxQueryParams is the bind parameter limit of the PostgreSQL extended query protocol.
const maxQueryParams = math.MaxUint16

var (
	errCreateMetadataQuery = errors.New("failed to create query for metadata")
	errCreatePayloadQuery  = errors.New("failed to create query for payload")
)

func GetNameQuery(name string) (string, string) {
	if name == "" {
		return "", ""
	}

	name = fmt.Sprintf(`%%%s%%`, strings.ToLower(name))
	nq := `LOWER(name) LIKE :name`

	return nq, name
}

func GetMetadataQuery(m map[string]any) (mb []byte, mq string, err error) {
	if len(m) > 0 {
		mq = `metadata @> :metadata`

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", errors.Wrap(err, errCreateMetadataQuery)
		}
		mb = b
	}
	return mb, mq, nil
}

func GetPayloadQuery(m map[string]any) (mb []byte, mq string, err error) {
	if len(m) > 0 {
		mq = `payload @> :payload`

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", errors.Wrap(err, errCreatePayloadQuery)
		}
		mb = b
	}
	return mb, mq, nil
}

// GetOrderQuery resolves order against fields, which maps API-facing order
// keys to real SQL column expressions (with LOWER() baked in where needed).
// Falls back to "id" for anything not in fields, so this is safe even if a
// caller forgets to validate order upstream.
func GetOrderQuery(order string, fields map[string]string) string {
	if col, ok := fields[order]; ok {
		return col
	}
	return "id"
}

func GetDirQuery(dir string) string {
	switch dir {
	case "asc":
		return "ASC"
	default:
		return "DESC"
	}
}

func GetOffsetLimitQuery(limit uint64) string {
	if limit != 0 {
		return "LIMIT :limit OFFSET :offset"
	}

	return ""
}


// GetLikeQuery returns a case-insensitive LIKE condition using a named parameter.
// Returns ("", "") if value is empty.
func GetLikeQuery(field, value string) (string, string) {
	if value == "" {
		return "", ""
	}
	return fmt.Sprintf("LOWER(%s) LIKE :%s", field, field), fmt.Sprintf("%%%s%%", strings.ToLower(value))
}

func GetGroupIDsQuery(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("group_id IN ('%s') ", strings.Join(ids, "','"))
}

func Total(ctx context.Context, db Database, query string, params any) (uint64, error) {
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

func BuildWhereClause(filters ...string) string {
	var queryFilters []string
	for _, filter := range filters {
		if filter != "" {
			queryFilters = append(queryFilters, filter)
		}
	}

	if len(queryFilters) > 0 {
		return fmt.Sprintf(" WHERE %s", strings.Join(queryFilters, " AND "))
	}

	return ""
}

type namedExecer interface {
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

// NamedExecBatch executes the named insert query in chunks within the bind parameter limit.
func NamedExecBatch[T any](ctx context.Context, db namedExecer, query string, rows []T) error {
	if len(rows) == 0 {
		return nil
	}

	_, params, err := sqlx.Named(query, rows[0])
	if err != nil {
		return err
	}

	for chunk := range slices.Chunk(rows, maxQueryParams/len(params)) {
		if _, err := db.NamedExecContext(ctx, query, chunk); err != nil {
			return err
		}
	}

	return nil
}
