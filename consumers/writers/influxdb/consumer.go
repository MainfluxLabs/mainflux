// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"context"
	"math"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	influxdb2write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

const senmlPoints = "messages"

var _ consumers.Consumer = (*influxRepo)(nil)

type RepoConfig struct {
	Bucket string
	Org    string
}
type influxRepo struct {
	client influxdb2.Client
	cfg    RepoConfig
}

// New returns new InfluxDB writer.
func New(client influxdb2.Client, config RepoConfig) consumers.Consumer {
	return &influxRepo{
		client: client,
		cfg:    config,
	}
}

func (repo *influxRepo) Consume(message interface{}) error {
	var err error
	var pts []*influxdb2write.Point
	switch m := message.(type) {
	case json.Messages:
		pts, err = repo.jsonPoints(m)
	default:
		pts, err = repo.senmlPoints(m)
	}
	if err != nil {
		return err
	}
	writeAPI := repo.client.WriteAPIBlocking(repo.cfg.Org, repo.cfg.Bucket)
	err = writeAPI.WritePoint(context.Background(), pts...)
	return err
}

func (repo *influxRepo) senmlPoints(messages interface{}) ([]*influxdb2write.Point, error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return nil, errors.ErrSaveMessage
	}
	var pts []*write.Point
	for _, msg := range msgs {
		tgs := map[string]string{
			"subtopic":  msg.Subtopic,
			"publisher": msg.Publisher,
			"name":      msg.Name,
		}

		flds := map[string]interface{}{
			"protocol":   msg.Protocol,
			"unit":       msg.Unit,
			"updateTime": msg.UpdateTime,
		}

		switch {
		case msg.Value != nil:
			flds["value"] = *msg.Value
		case msg.StringValue != nil:
			flds["stringValue"] = *msg.StringValue
		case msg.DataValue != nil:
			flds["dataValue"] = *msg.DataValue
		case msg.BoolValue != nil:
			flds["boolValue"] = *msg.BoolValue
		}

		if msg.Sum != nil {
			flds["sum"] = *msg.Sum
		}

		sec, dec := math.Modf(msg.Time)
		t := time.Unix(int64(sec), int64(dec*(1e9)))

		pt := influxdb2.NewPoint(senmlPoints, tgs, flds, t)
		pts = append(pts, pt)
	}

	return pts, nil
}

func (repo *influxRepo) jsonPoints(msgs json.Messages) ([]*influxdb2write.Point, error) {
	var pts []*write.Point
	for i, m := range msgs.Data {
		t := time.Unix(0, m.Created+int64(i))

		flat, err := json.Flatten(m.Payload)
		if err != nil {
			return nil, errors.Wrap(json.ErrTransform, err)
		}
		m.Payload = flat

		// Copy first-level fields so that the original Payload is unchanged.
		fields := make(map[string]interface{})
		for k, v := range m.Payload {
			fields[k] = v
		}
		// At least one known field need to exist so that COUNT can be performed.
		fields["protocol"] = m.Protocol

		tags := map[string]string{
			"subtopic":  m.Subtopic,
			"publisher": m.Publisher,
		}
		pt := influxdb2.NewPoint(msgs.Format, tags, fields, t)
		pts = append(pts, pt)
	}

	return pts, nil
}
