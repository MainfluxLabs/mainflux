// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"

	writer "github.com/MainfluxLabs/mainflux/consumers/writers/influxdb"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const valueFields = 5

var (
	testLog, _    = log.New(os.Stdout, log.Info.String())
	streamsSize   = 250
	rowCountSenml = fmt.Sprintf(`from(bucket: "%s")
	|> range(start: -1h, stop: 1h)
	|> filter(fn: (r) => r["_measurement"] == "messages")
	|> filter(fn: (r) => r["_field"] == "dataValue" or r["_field"] == "stringValue" or r["_field"] == "value" or r["_field"] == "boolValue" or r["_field"] == "sum" )
	|> group(columns: ["_measurement"])
	|> count()
	|> yield(name: "count")`, repoCfg.Bucket)

	rowCountJson = fmt.Sprintf(`from(bucket: "%s")
	|> range(start: -1h, stop: 1h)
	|> filter(fn: (r) => r["_measurement"] == "some_json")
	|> filter(fn: (r) => r["_field"] == "field_1" or r["_field"] == "field_2" or r["_field"] == "field_3" or r["_field"] == "field_4" or r["_field"] == "field_5/field_1" or r["_field"] == "field_5/field_2")
	|> count()
	|> yield(name: "count")`, repoCfg.Bucket)
	subtopic = "topic"

	client  influxdb2.Client
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
	repoCfg         = writer.RepoConfig{
		Bucket: dbBucket,
		Org:    dbOrg,
	}
	errUnexpectedType = errors.New("Unexpected response type")

	idProvider = uuid.New()
)

func deleteBucket() error {
	bucketsAPI := client.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(context.Background(), repoCfg.Bucket)
	if err != nil {
		return err
	}

	if err = bucketsAPI.DeleteBucket(context.Background(), bucket); err != nil {
		return err
	}

	return nil
}
func createBucket() error {
	orgAPI := client.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(context.Background(), repoCfg.Org)
	if err != nil {
		return err
	}
	bucketsAPI := client.BucketsAPI()
	if _, err = bucketsAPI.CreateBucketWithName(context.Background(), org, repoCfg.Bucket); err != nil {
		return err
	}

	return nil
}
func resetBucket() error {
	if err := deleteBucket(); err != nil {
		return err
	}
	if err := createBucket(); err != nil {
		return err
	}

	return nil
}

func queryDB(fluxQuery string) (int, error) {
	rowCount := 0
	queryAPI := client.QueryAPI(repoCfg.Org)

	// get QueryTableResult
	result, err := queryAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return rowCount, err
	}
	if result.Next() {
		value, ok := result.Record().Value().(int64)
		if !ok {
			return rowCount, errUnexpectedType
		}
		rowCount = int(value)
	}
	if result.Err() != nil {
		return rowCount, result.Err()
	}

	return rowCount, nil
}

func TestSaveSenML(t *testing.T) {
	repo := writer.New(client, repoCfg)
	cases := []struct {
		desc         string
		msgsNum      int
		expectedSize int
	}{
		{
			desc:         "save a single message",
			msgsNum:      1,
			expectedSize: 1,
		},
		{
			desc:         "save a batch of messages",
			msgsNum:      streamsSize,
			expectedSize: streamsSize,
		},
	}

	for _, tc := range cases {
		err := resetBucket()
		require.Nil(t, err, fmt.Sprintf("Cleaning data from InfluxDB expected to succeed: %s.\n", err))

		now := time.Now().UnixNano()
		var msgs []senml.Message

		pubID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		for i := 0; i < tc.msgsNum; i++ {
			msg := senml.Message{
				Publisher:  pubID,
				Protocol:   "http",
				Name:       "test name",
				Unit:       "km",
				UpdateTime: 5456565466,
			}
			// Mix possible values as well as value sum.
			count := i % valueFields
			switch count {
			case 0:
				msg.Subtopic = subtopic
				msg.Value = &v
			case 1:
				msg.BoolValue = &boolV
			case 2:
				msg.StringValue = &stringV
			case 3:
				msg.DataValue = &dataV
			case 4:
				msg.Sum = &sum
			}

			msg.Time = float64(now)/float64(1e9) - float64(i)
			msgs = append(msgs, msg)
		}

		err = repo.Consume(msgs)
		assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))
		count, err := queryDB(rowCountSenml)
		assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data expected to succeed: %s.\n", err))
		assert.Equal(t, tc.expectedSize, count, fmt.Sprintf("Expected to have %d messages saved, found %d instead.\n", tc.expectedSize, count))
	}
}

func TestSaveJSON(t *testing.T) {
	repo := writer.New(client, repoCfg)

	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
		Publisher: pubID,
		Created:   time.Now().UnixNano(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "mqtt",
		Payload: map[string]interface{}{
			"field_1": 123,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42,
			},
		},
	}

	invalidKeySepMsg := msg
	invalidKeySepMsg.Payload = map[string]interface{}{
		"field_1": 123,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]interface{}{
			"field_1": "value",
			"field_2": 42,
		},
		"field_6/field_7": "value",
	}
	invalidKeyNameMsg := msg
	invalidKeyNameMsg.Payload = map[string]interface{}{
		"field_1": 123,
		"field_2": "value",
		"field_3": false,
		"field_4": 12.344,
		"field_5": map[string]interface{}{
			"field_1": "value",
			"field_2": 42,
		},
		"publisher": "value",
	}

	now := time.Now().UnixNano()
	msgs := json.Messages{
		Format: "some_json",
	}
	invalidKeySepMsgs := json.Messages{
		Format: "some_json",
	}
	invalidKeyNameMsgs := json.Messages{
		Format: "some_json",
	}

	for i := 0; i < streamsSize; i++ {
		msg.Created = now
		msgs.Data = append(msgs.Data, msg)
		invalidKeySepMsgs.Data = append(invalidKeySepMsgs.Data, invalidKeySepMsg)
		invalidKeyNameMsgs.Data = append(invalidKeyNameMsgs.Data, invalidKeyNameMsg)
	}

	cases := []struct {
		desc string
		msgs json.Messages
		err  error
	}{
		{
			desc: "consume valid json messages",
			msgs: msgs,
			err:  nil,
		},
		{
			desc: "consume invalid json messages containing invalid key separator",
			msgs: invalidKeySepMsgs,
			err:  json.ErrInvalidKey,
		},
		{
			desc: "consume invalid json messages containing invalid key name",
			msgs: invalidKeySepMsgs,
			err:  json.ErrInvalidKey,
		},
	}
	for _, tc := range cases {
		err := resetBucket()
		require.Nil(t, err, fmt.Sprintf("Cleaning data from InfluxDB expected to succeed: %s.\n", err))

		switch err = repo.Consume(tc.msgs); err {
		case nil:
			count, err := queryDB(rowCountJson)
			assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data expected to succeed: %s.\n", err))
			assert.Equal(t, streamsSize, count, fmt.Sprintf("Expected to have %d messages saved, found %d instead.\n", streamsSize, count))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		}
	}
}
