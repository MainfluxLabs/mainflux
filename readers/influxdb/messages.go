package influxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	jsont "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

const (
	// Measurement for SenML messages
	defMeasurement = "messages"
	// noLimit is used to indicate that there is no limit for the number of results.
	noLimit = 0
)

var _ readers.MessageRepository = (*influxRepository)(nil)

var errResultTime = errors.New("invalid result time")

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

func (repo *influxRepository) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *influxRepository) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *influxRepository) Restore(ctx context.Context, messages ...senml.Message) error {
	pts, err := repo.senmlPoints(messages)
	if err != nil {
		return err
	}

	writeAPI := repo.client.WriteAPIBlocking(repo.cfg.Org, repo.cfg.Bucket)
	err = writeAPI.WritePoint(context.Background(), pts...)

	return nil
}

func (repo *influxRepository) senmlPoints(messages []senml.Message) ([]*write.Point, error) {
	var pts []*write.Point
	for _, msg := range messages {
		tgs, flds := senmlTags(msg), senmlFields(msg)

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt := influxdb2.NewPoint(defMeasurement, tgs, flds, t)
		pts = append(pts, pt)
	}

	return pts, nil
}

func (repo *influxRepository) readAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defMeasurement
	if rpm.Format != "" {
		format = rpm.Format
	}

	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	var sb strings.Builder

	condition, timeRange := fmtCondition(rpm)
	sb.WriteString(`import "influxdata/influxdb/schema"`)
	sb.WriteString(fmt.Sprintf(`from(bucket: "%s")`, repo.cfg.Bucket))
	// FluxQL syntax requires timeRange filter in this position, do not change.
	sb.WriteString(timeRange)
	// This is required to get messsage structure. Otherwise query returns fields in seperate rows.
	sb.WriteString(` |> schema.fieldsAsCols() `)
	sb.WriteString(`|> group()`)
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._measurement == "%s")`, format))
	sb.WriteString(condition)
	sb.WriteString(`|> sort(columns: ["_time"], desc: true)`)
	if rpm.Limit != noLimit {
		sb.WriteString(fmt.Sprintf(`|> limit(n:%d,offset:%d)`, rpm.Limit, rpm.Offset))
	}
	sb.WriteString(`|> yield(name: "sort")`)
	query := sb.String()
	resp, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

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

	total, err := repo.count(format, condition, timeRange)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Total:        total, //total,
		Messages:     messages,
	}

	return page, nil
}

func (repo *influxRepository) count(measurement, condition string, timeRange string) (uint64, error) {

	var sb strings.Builder
	sb.WriteString(`import "influxdata/influxdb/schema" `)
	sb.WriteString(fmt.Sprintf(`from(bucket: "%s")`, repo.cfg.Bucket))
	sb.WriteString(timeRange)
	sb.WriteString(` |> schema.fieldsAsCols() `)
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r._measurement == "%s")`, measurement))
	sb.WriteString(condition)
	sb.WriteString(`|> group()`)
	sb.WriteString(`|> count(column:"_measurement")`)
	sb.WriteString(`|> yield(name: "count")`)

	cmd := sb.String()
	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	resp, err := queryAPI.Query(context.Background(), cmd)

	if err != nil {
		return 0, err
	}

	switch resp.Next() {
	case true:
		valueMap := resp.Record().Values()

		val, ok := valueMap["_measurement"].(int64)
		if !ok {
			return 0, nil
		}
		return uint64(val), nil

	default:
		// same as no rows.
		return 0, nil
	}

}

func fmtCondition(rpm readers.PageMetadata) (string, string) {
	// TODO: adapt filters to flux
	var timeRange string

	var sb strings.Builder

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
		fromValue := int64(value.(float64)*1e9) - 1
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
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.boolValue == %v)`, value))
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
		case "channel", "created", "subtopic", "publisher", "protocol":
			ret[name] = field
		case "_time":
			name = "time"
			t, ok := field.(time.Time)
			if !ok {
				return nil, errResultTime
			}
			v := float64(t.UnixNano()) / 1e9
			ret[name] = v
			continue
		case "table", "_start", "_stop", "result", "_measurement":
			break
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
