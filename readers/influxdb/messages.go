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

	condition, timeRange := fmtCondition(chanID, rpm)
	sb.WriteString(`import "influxdata/influxdb/v1"`)
	sb.WriteString(fmt.Sprintf(`from(bucket: "%s")`, repo.cfg.Bucket))
	// FluxQL syntax requires timeRange filter in this position, do not change.
	sb.WriteString(timeRange)
	// This is required to get messsage structure. Otherwise query returns fields in seperate rows.
	sb.WriteString(`|> v1.fieldsAsCols()`)
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._measurement == "%s")`, format))
	sb.WriteString(condition)
	sb.WriteString(fmt.Sprintf(`|> limit(n:%d,offset:%d)`, rpm.Limit, rpm.Offset))
	sb.WriteString(`|> sort(columns: ["_time"], desc: true)`)
	sb.WriteString(`|> yield(name: "sort")`)
	query := sb.String()
	resp, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	/*
		if len(resp.Results) == 0 || len(resp.Results[0].Series) == 0 {
			return readers.MessagesPage{}, nil
		}
	*/
	var messages []readers.Message

	var valueMap map[string]interface{}
	for resp.Next() {
		valueMap = resp.Record().Values()
		msg, err := parseMessage(format, valueMap)
		if err != nil {
			return readers.MessagesPage{}, err
		}
		messages = append(messages, msg)
	}
	if resp.Err() != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, resp.Err())
	}

	fmt.Println("Query: ", query)
	/*
		total, err := repo.count(format, condition)
		if err != nil {
			return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
	*/
	total := uint64(len(messages))

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Total:        total, //total,
		Messages:     messages,
	}

	return page, nil

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
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r["channel"] == "%s" )`, chanID))

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return sb.String(), timeRange
	}

	if err := json.Unmarshal(meta, &query); err != nil {
		return sb.String(), timeRange
	}

	//range(start:...) is a must for FluxQL syntax
	from := `start: time(v:0)`
	if value, ok := query["from"]; ok {
		fromValue := int64(value.(float64) * 1e9)
		from = fmt.Sprintf(`start: time(v: %d )`, fromValue)
	}
	//range(...,stop:) is an option for FluxQL syntax
	to := ""
	if value, ok := query["to"]; ok {
		toValue := int64(value.(float64) * 1e9)
		to = fmt.Sprintf(`, stop: time(v: %d )`, toValue)
	}
	// timeRange returned seperately because
	// in FluxQL time range must be at the
	// beginning of the query.
	timeRange = fmt.Sprintf(`|> range(%s %s)`, from, to)

	for name, value := range query {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.%s == "%s" )`, name, value))
		case "v":
			comparator := readers.ParseValueComparator(query)
			//flux eq comparator is different
			if comparator == "=" {
				comparator = "=="
			}
			sb.WriteString(`|> filter(fn: (r) => exists r.value)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.value %s %v)`, comparator, value))
		case "vb":
			sb.WriteString(`|> filter(fn: (r) => exists r.boolValue)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.boolValue == "%s")`, value))
		case "vs":
			sb.WriteString(`|> filter(fn: (r) => exists r.stringValue)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.stringValue == "%s")`, value))
		case "vd":
			sb.WriteString(`|> filter(fn: (r) => exists r.dataValue)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.dataValue == "%s")`, value))
		}
	}

	return sb.String(), timeRange
}

func parseMessage(measurement string, valueMap map[string]interface{}) (interface{}, error) {
	switch measurement {
	case defMeasurement:
		return parseSenml(valueMap)
	default:
		return parseJSON(valueMap)
	}
}

func underscore(name string) string {
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
	return string(buff)
}

func parseSenml(valueMap map[string]interface{}) (interface{}, error) {
	msg := make(map[string]interface{})

	for k, v := range valueMap {
		k = underscore(k)
		if k == "_time" {
			k = "time"
			t, ok := v.(time.Time)
			if !ok {
				return nil, errResultTime
			}
			v := float64(t.UnixNano()) / 1e9
			msg[k] = v
			continue
		}
		msg[k] = v
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	senmlMsg := senml.Message{}
	if err := json.Unmarshal(data, &senmlMsg); err != nil {
		return nil, err
	}
	return senmlMsg, nil
}

func parseJSON(valueMap map[string]interface{}) (interface{}, error) {
	ret := make(map[string]interface{})
	pld := make(map[string]interface{})
	for name, field := range valueMap {
		switch name {
		case "channel", "created", "subtopic", "publisher", "protocol", "time":
			ret[name] = field
		default:
			v := field
			if val, ok := v.(json.Number); ok {
				var err error
				v, err = val.Float64()
				if err != nil {
					return nil, err
				}
			}
			pld[name] = v
		}
	}
	ret["payload"] = jsont.ParseFlat(pld)
	return ret, nil
}
