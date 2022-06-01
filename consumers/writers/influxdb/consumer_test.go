// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"

	"github.com/gofrs/uuid"
	writer "github.com/mainflux/mainflux/consumers/writers/influxdb"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

const valueFields = 5

var (
	port        string
	testLog, _  = log.New(os.Stdout, log.Info.String())
	streamsSize = 250

	selectMsgs = fmt.Sprintf(`from(bucket: "%s")
	|> range(start: v.timeRangeStart, stop: v.timeRangeStop)
	|> filter(fn: (r) => r["_measurement"] == "messages")
	|> sort()
	|> yield(name: "sort")`, repoCfg.Bucket)
	client   influxdb2.Client
	subtopic = "topic"

	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
	repoCfg         = writer.RepoConfig{
		Bucket: dbBucket,
		Org:    dbOrg,
	}
)

func deleteBucket() error {
	bucketsAPI := client.BucketsAPI()
	bucket, err := bucketsAPI.FindBucketByName(context.Background(), repoCfg.Bucket)
	if err != nil {
		return err
	}
	err = bucketsAPI.DeleteBucket(context.Background(), bucket)
	if err != nil {
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
	_, err = bucketsAPI.CreateBucketWithName(context.Background(), org, repoCfg.Bucket)
	if err != nil {
		return err
	}
	return nil
}

func resetBucket() error {
	err := deleteBucket()
	if err != nil {
		return err
	}
	err = createBucket()
	if err != nil {
		return err
	}
	return nil
}

func queryDB(fluxQuery string) (error, int) {
	queryAPI := client.QueryAPI(repoCfg.Org)
	rowCount := 0
	// get QueryTableResult
	result, err := queryAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return err, rowCount
	}

	// Iterate over query response
	for result.Next() {
		// Notice when group key has changed
		if result.TableChanged() {
			fmt.Printf("table: %s\n", result.TableMetadata().String())
		}
		rowCount++
		// Access data
		fmt.Printf("value: %v\n", result.Record().Value())
	}
	// check for an error
	if result.Err() != nil {
		return result.Err(), 0
	}
	return nil, rowCount
}

func TestSaveSenml(t *testing.T) {

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
		// Clean previously saved messages.
		err := resetBucket()
		require.Nil(t, err, fmt.Sprintf("Cleaning data from InfluxDB expected to succeed: %s.\n", err))

		now := time.Now().UnixNano()
		msg := senml.Message{
			Channel:    "45",
			Publisher:  "2580",
			Protocol:   "http",
			Name:       "test name",
			Unit:       "km",
			UpdateTime: 5456565466,
		}
		var msgs []senml.Message

		for i := 0; i < tc.msgsNum; i++ {
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

			msg.Time = float64(now)/float64(1e9) + float64(i)
			msgs = append(msgs, msg)
		}

		err = repo.Consume(msgs)
		assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

		err, count := queryDB(selectMsgs)

		assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data expected to succeed: %s.\n", err))
		assert.Equal(t, tc.expectedSize, count, fmt.Sprintf("Expected to have %d messages saved, found %d instead.\n", tc.expectedSize, count))
	}
}

func TestSaveJSON(t *testing.T) {
	repo := writer.New(client, repoCfg)

	chid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
		Channel:   chid.String(),
		Publisher: pubid.String(),
		Created:   time.Now().Unix(),
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

	now := time.Now().Unix()
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
		msg.Created = now + int64(i)
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
		err = repo.Consume(tc.msgs)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))

		err, count := queryDB(selectMsgs)
		assert.Nil(t, err, fmt.Sprintf("Querying InfluxDB to retrieve data expected to succeed: %s.\n", err))
		assert.Equal(t, streamsSize, count, fmt.Sprintf("Expected to have %d messages saved, found %d instead.\n", streamsSize, count))
	}
}
