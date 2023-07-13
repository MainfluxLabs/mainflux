package dbutil

import (
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

func GetOwnerQuery(owner, ownerDbId string) string {
	if owner == "" {
		return ""
	}

	return fmt.Sprintf("%s = :%s", ownerDbId, ownerDbId)
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
