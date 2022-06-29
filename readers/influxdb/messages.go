package influxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	jsont "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

const (
	countCol = "count_protocol"
	// Measurement for SenML messages
	defMeasurement = "messages"
)

var _ readers.MessageRepository = (*influxRepository)(nil)

var (
	errResultSet  = errors.New("invalid result set")
	errResultTime = errors.New("invalid result time")
)

type RepoConfig struct {
	Bucket string
	Org    string
}
type influxRepository struct {
	cfg    RepoConfig
	client influxdb2.Client
}

// New returns new InfluxDB reader.
func New(client influxdb2.Client, repoCfg RepoConfig) readers.MessageRepository {
	return &influxRepository{
		repoCfg,
		client,
	}
}

func (repo *influxRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {

	format := defMeasurement
	if rpm.Format != "" {
		format = rpm.Format
	}

	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	var sb strings.Builder
	//TODO: correct contition queries as flux queries
	condition, timeRange := fmtCondition(chanID, rpm)

	sb.WriteString(fmt.Sprintf(`from(bucket: "%s")`, repo.cfg.Bucket))
	// FluxQL syntax requires timeRange filter in this position.
	sb.WriteString(timeRange)
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._measurement == "%s")`, format))
	sb.WriteString(condition)
	sb.WriteString(fmt.Sprintf(`|> limit(n:%d,offset:%d)`, rpm.Limit, rpm.Offset))
	sb.WriteString(`|> sort(columns: ["_time"], desc: true)`)
	sb.WriteString(`|> yield(name: "sort")`)
	//TODO: parse response values into messages.
	_, err := queryAPI.Query(context.Background(), sb.String())
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	//if resp.Error() != nil {
	//	return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, resp.Error())
	//}

	//if len(resp.Results) == 0 || len(resp.Results[0].Series) == 0 {
	//	return readers.MessagesPage{}, nil
	//}

	//var messages []readers.Message
	//result := resp.Results[0].Series[0]
	/*
		for _, v := range result.Values {
			msg, err := parseMessage(format, result.Columns, v)
			if err != nil {
				return readers.MessagesPage{}, err
			}
			messages = append(messages, msg)
		}

		total, err := repo.count(format, condition)
		if err != nil {
			return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}

		page := readers.MessagesPage{
			PageMetadata: rpm,
			Total:        total,
			Messages:     messages,
		}

		return page, nil
	*/
	return readers.MessagesPage{}, nil
}

func (repo *influxRepository) count(measurement, condition string) (uint64, error) {
	// TODO: Adapt this base query to flux
	cmd := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, measurement, condition)
	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	_, err := queryAPI.Query(context.Background(), cmd)

	if err != nil {
		return 0, err
	}
	//TODO: Adapt response of influxdb2 and get rowcount
	// if resp.Error() != nil {
	// 	return 0, resp.Error()
	// }

	// if len(resp.Results) == 0 ||
	// 	len(resp.Results[0].Series) == 0 ||
	// 	len(resp.Results[0].Series[0].Values) == 0 {
	// 	return 0, nil
	// }

	// countIndex := 0
	// for i, col := range resp.Results[0].Series[0].Columns {
	// 	if col == countCol {
	// 		countIndex = i
	// 		break
	// 	}
	// }

	// result := resp.Results[0].Series[0].Values[0]
	// if len(result) < countIndex+1 {
	// 	return 0, nil
	// }

	// count, ok := result[countIndex].(json.Number)
	// if !ok {
	// 	return 0, nil
	// }
	// return strconv.ParseUint(count.String(), 10, 64)
	return 10, nil
}

func fmtCondition(chanID string, rpm readers.PageMetadata) (string, string) {
	// TODO: adapt filters to flux
	var timeRange string

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r["channel"] == "%s")`, chanID))

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return sb.String(), timeRange
	}

	if err := json.Unmarshal(meta, &query); err != nil {
		return sb.String(), timeRange
	}

	for name, value := range query {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r["%s"] == "%s"`, name, value))
		case "protocol":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._field == "protocol" and r._value == "%s")`, value))
		case "v":
			//comparator := readers.ParseValueComparator(query)
			//TODO: adapt comparator. flux comparators are different
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._field == "value" and r._value == "%s")`, value))
		case "vb":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._field == "boolValue" and r._value == "%s")`, value))
		case "vs":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._field == "stringValue" and r._value == "%s")`, value))
		case "vd":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._field == "dataValue" and r._value == "%s")`, value))

		}
		//range(start: ... ) is a must for FluxQL syntax
		timeRangeTo := ""
		if value, ok := query["to"]; ok {
			to := int64(value.(float64) * 1e9)
			timeRangeTo = fmt.Sprintf(`, stop: time(v:%d)`, to)
		}

		switch value, ok := query["from"]; ok {
		case true:
			from := int64(value.(float64) * 1e9)
			timeRange = fmt.Sprintf(`|> range(start: time(v:%d) %s)`, from, timeRangeTo)
		default:
			from := 0
			timeRange = fmt.Sprintf(`|> range(start: time(v:%d) %s)`, from, timeRangeTo)
		}

	}
	return sb.String(), timeRange
}

func parseMessage(measurement string, names []string, fields []interface{}) (interface{}, error) {
	switch measurement {
	case defMeasurement:
		return parseSenml(names, fields)
	default:
		return parseJSON(names, fields)
	}
}

func underscore(names []string) {
	for i, name := range names {
		var buff []rune
		idx := 0
		for i, c := range name {
			if unicode.IsUpper(c) {
				buff = append(buff, []rune(name[idx:i])...)
				buff = append(buff, []rune{'_', unicode.ToLower(c)}...)
				idx = i + 1
				continue
			}
		}
		buff = append(buff, []rune(name[idx:])...)
		names[i] = string(buff)
	}
}

func parseSenml(names []string, fields []interface{}) (interface{}, error) {
	m := make(map[string]interface{})
	if len(names) > len(fields) {
		return nil, errResultSet
	}
	underscore(names)
	for i, name := range names {
		if name == "time" {
			val, ok := fields[i].(string)
			if !ok {
				return nil, errResultTime
			}
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return nil, err
			}
			v := float64(t.UnixNano()) / 1e9
			m[name] = v
			continue
		}
		m[name] = fields[i]
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	msg := senml.Message{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func parseJSON(names []string, fields []interface{}) (interface{}, error) {
	ret := make(map[string]interface{})
	pld := make(map[string]interface{})
	for i, n := range names {
		switch n {
		case "channel", "created", "subtopic", "publisher", "protocol", "time":
			ret[n] = fields[i]
		default:
			v := fields[i]
			if val, ok := v.(json.Number); ok {
				var err error
				v, err = val.Float64()
				if err != nil {
					return nil, err
				}
			}
			pld[n] = v
		}
	}
	ret["payload"] = jsont.ParseFlat(pld)
	return ret, nil
}
