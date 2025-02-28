package dbutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var errCreateMetadataQuery = errors.New("failed to create query for metadata")

func GetNameQuery(name string) (string, string) {
	if name == "" {
		return "", ""
	}

	name = fmt.Sprintf(`%%%s%%`, strings.ToLower(name))
	nq := `LOWER(name) LIKE :name`

	return nq, name
}

func GetMetadataQuery(db string, m map[string]interface{}) (mb []byte, mq string, err error) {
	if len(m) > 0 {
		mq = `metadata @> :metadata`
		if db != "" {
			mq = db + "." + mq
		}

		b, err := json.Marshal(m)
		if err != nil {
			return nil, "", errors.Wrap(err, errCreateMetadataQuery)
		}
		mb = b
	}
	return mb, mq, nil
}

func GetOrderQuery(order string) string {
	switch order {
	case "name":
		return "name"
	default:
		return "id"
	}
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

func Total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
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
