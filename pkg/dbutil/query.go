package dbutil

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

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

func GetOrderQuery(order string) string {
	switch order {
	case "name":
		return "LOWER(name)"
	case "email":
		return "LOWER(email)"
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
