// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/stretchr/testify/assert"
)

func TestFormatMessage(t *testing.T) {
	const deviceTime = int64(1638310819000000000)

	pc := domain.PubConfigInfo{
		ThingID: "thing-1",
		ProfileConfig: &domain.ProfileConfig{
			ContentType: messaging.JSONContentType,
			Transformer: domain.Transformer{
				TimeField:    "ts",
				TimeFormat:   "unix",
				TimeLocation: "UTC",
			},
		},
	}

	objectMsg := protomfx.Message{
		Records: []*protomfx.Record{{Payload: []byte(`{"ts":1638310819,"temp":20}`)}},
	}

	cases := []struct {
		desc    string
		msg     protomfx.Message
		created int64
	}{
		{
			desc:    "format JSON object with time field",
			msg:     objectMsg,
			created: deviceTime,
		},
	}

	for _, tc := range cases {
		err := messaging.FormatMessage(pc, &tc.msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		createdMsg := fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.created, tc.msg.Created)
		assert.Equal(t, tc.created, tc.msg.Created, createdMsg)
	}
}

func TestToPayload(t *testing.T) {
	first := []byte(`{"key":"val1"}`)
	second := []byte(`{"key":"val2"}`)

	cases := []struct {
		desc string
		msg  protomfx.Message
		res  []byte
	}{
		{
			desc: "get payload of message with single record",
			msg: protomfx.Message{
				ContentType: messaging.JSONContentType,
				Records:     []*protomfx.Record{{Payload: first}},
			},
			res: first,
		},
		{
			desc: "get payload of message with multiple records",
			msg: protomfx.Message{
				ContentType: messaging.JSONContentType,
				Records:     []*protomfx.Record{{Payload: first}, {Payload: second}},
			},
			res: []byte(`[{"key":"val1"},{"key":"val2"}]`),
		},
		{
			desc: "get payload of SenML message with single record",
			msg: protomfx.Message{
				ContentType: messaging.SenMLContentType,
				Records:     []*protomfx.Record{{Payload: first}},
			},
			res: []byte(`[{"key":"val1"}]`),
		},
	}

	for _, tc := range cases {
		res, err := messaging.ToPayload(tc.msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.res, res))
	}
}

func TestToJSONMessages(t *testing.T) {
	msg := protomfx.Message{
		ThingId:  "thing-1",
		Subtopic: "subtopic",
		Protocol: "http",
		Created:  1,
		Records: []*protomfx.Record{
			{
				Payload: []byte(`{"key":"val1"}`),
				Time:    2,
			},
			{
				Payload: []byte(`{"key":"val2"}`),
				Time:    3,
			},
		},
	}

	jsonMsg := mfjson.Message{
		ThingID:  msg.ThingId,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
	}

	firstRes := jsonMsg
	firstRes.Payload = msg.Records[0].Payload
	firstRes.Created = msg.Records[0].Time

	secondRes := jsonMsg
	secondRes.Payload = msg.Records[1].Payload
	secondRes.Created = msg.Records[1].Time

	res := messaging.ToJSONMessages(msg)
	expected := []mfjson.Message{firstRes, secondRes}
	assert.Equal(t, expected, res, fmt.Sprintf("expected %v got %v\n", expected, res))
}

func TestToSenMLMessages(t *testing.T) {
	msg := protomfx.Message{
		ThingId:  "thing-1",
		Subtopic: "subtopic",
		Protocol: "http",
		Records: []*protomfx.Record{
			{
				Payload: []byte(`{"name":"temp","time":2}`),
				Time:    2,
			},
			{
				Payload: []byte(`{"name":"hum","time":3}`),
				Time:    3,
			},
		},
	}

	invalidMsg := msg
	invalidMsg.Records = []*protomfx.Record{{Payload: []byte(`invalid`)}}

	senmlMsg := senml.Message{
		ThingID:  msg.ThingId,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
	}

	tempRes := senmlMsg
	tempRes.Name = "temp"
	tempRes.Time = 2

	humRes := senmlMsg
	humRes.Name = "hum"
	humRes.Time = 3

	cases := []struct {
		desc string
		msg  protomfx.Message
		res  []senml.Message
		err  bool
	}{
		{
			desc: "convert message with records",
			msg:  msg,
			res:  []senml.Message{tempRes, humRes},
			err:  false,
		},
		{
			desc: "convert message with invalid record payload",
			msg:  invalidMsg,
			res:  nil,
			err:  true,
		},
	}

	for _, tc := range cases {
		res, err := messaging.ToSenMLMessages(tc.msg)
		assert.Equal(t, tc.err, err != nil, fmt.Sprintf("%s: unexpected error %v\n", tc.desc, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.res, res))
	}
}
