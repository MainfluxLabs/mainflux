// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging_test

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/stretchr/testify/assert"
)

func TestToJSONMessages(t *testing.T) {
	msg := protomfx.Message{
		ThingId:  "thing-1",
		Subtopic: "subtopic",
		Protocol: "http",
		Payload:  []byte(`[{"key":"val1"},{"key":"val2"}]`),
		Created:  1,
	}

	recordsMsg := msg
	recordsMsg.Records = []*protomfx.Record{
		{Payload: []byte(`{"key":"val1"}`), Created: 2},
		{Payload: []byte(`{"key":"val2"}`), Created: 3},
	}

	jsonMsg := mfjson.Message{
		ThingID:  msg.ThingId,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
	}

	msgRes := jsonMsg
	msgRes.Payload = msg.Payload
	msgRes.Created = msg.Created

	firstRes := jsonMsg
	firstRes.Payload = recordsMsg.Records[0].Payload
	firstRes.Created = recordsMsg.Records[0].Created

	secondRes := jsonMsg
	secondRes.Payload = recordsMsg.Records[1].Payload
	secondRes.Created = recordsMsg.Records[1].Created

	cases := []struct {
		desc string
		msg  protomfx.Message
		res  []mfjson.Message
	}{
		{
			desc: "convert message without records",
			msg:  msg,
			res:  []mfjson.Message{msgRes},
		},
		{
			desc: "convert message with records",
			msg:  recordsMsg,
			res:  []mfjson.Message{firstRes, secondRes},
		},
	}

	for _, tc := range cases {
		res := messaging.ToJSONMessages(tc.msg)
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.res, res))
	}
}
