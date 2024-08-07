// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	mfsenml "github.com/MainfluxLabs/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformJSON(t *testing.T) {
	// Following hex-encoded bytes correspond to the content of:
	// [{-2: "base-name", -3: 100.0, -4: "base-unit", -1: 10, -5: 10.0, -6: 100.0, 0: "name", 1: "unit", 6: 300.0, 7: 150.0, 2: 42.0, 5: 10.0}]
	// For more details for mapping SenML labels to integers, please take a look here: https://tools.ietf.org/html/rfc8428#page-19.
	jsonBytes, err := hex.DecodeString("5b7b22626e223a22626173652d6e616d65222c226274223a3130302c226275223a22626173652d756e6974222c2262766572223a31302c226276223a31302c226273223a3130302c226e223a226e616d65222c2275223a22756e6974222c2274223a3330302c227574223a3135302c2276223a34322c2273223a31307d5d")
	require.Nil(t, err, "Decoding JSON expected to succeed")

	tr := senml.New()
	msg := protomfx.Message{
		Channel:   "channel",
		Subtopic:  "subtopic",
		Publisher: "publisher",
		Protocol:  "protocol",
		Payload:   jsonBytes,
		Profile:   &protomfx.Profile{ContentType: senml.JSON},
	}

	// 82AD2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D650164756E697406F95CB0036331323307F958B002F9514005F94900AA2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D6506F95CB007F958B005F94900

	jsonPld := msg
	jsonPld.Payload = jsonBytes

	val := 52.0
	sum := 110.0
	msgs := []senml.Message{
		{
			Subtopic:   "subtopic",
			Publisher:  "publisher",
			Protocol:   "protocol",
			Name:       "base-namename",
			Unit:       "unit",
			Time:       400,
			UpdateTime: 150,
			Value:      &val,
			Sum:        &sum,
		},
	}

	cases := []struct {
		desc string
		msg  protomfx.Message
		msgs interface{}
		err  error
	}{
		{
			desc: "test normalize JSON",
			msg:  jsonPld,
			msgs: msgs,
			err:  nil,
		},
		{
			desc: "test normalize defaults to JSON",
			msg:  msg,
			msgs: msgs,
			err:  nil,
		},
	}

	for _, tc := range cases {
		msgs, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.msgs, msgs, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.msgs, msgs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

func TestTransformCBOR(t *testing.T) {
	// Following hex-encoded bytes correspond to the content of:
	// [{-2: "base-name", -3: 100.0, -4: "base-unit", -1: 10, -5: 10.0, -6: 100.0, 0: "name", 1: "unit", 6: 300.0, 7: 150.0, 2: 42.0, 5: 10.0}]
	// For more details for mapping SenML labels to integers, please take a look here: https://tools.ietf.org/html/rfc8428#page-19.
	cborBytes, err := hex.DecodeString("81ac2169626173652d6e616d6522fb40590000000000002369626173652d756e6974200a24fb402400000000000025fb405900000000000000646e616d650164756e697406fb4072c0000000000007fb4062c0000000000002fb404500000000000005fb4024000000000000")
	require.Nil(t, err, "Decoding CBOR expected to succeed")

	tooManyBytes, err := hex.DecodeString("82AD2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D650164756E697406F95CB0036331323307F958B002F9514005F94900AA2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D6506F95CB007F958B005F94900")
	require.Nil(t, err, "Decoding CBOR expected to succeed")

	tr := senml.New()
	msg := protomfx.Message{
		Channel:   "channel",
		Subtopic:  "subtopic",
		Publisher: "publisher",
		Protocol:  "protocol",
		Payload:   cborBytes,
		Profile:   &protomfx.Profile{ContentType: senml.CBOR},
	}

	// 82AD2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D650164756E697406F95CB0036331323307F958B002F9514005F94900AA2169626173652D6E616D6522F956402369626173652D756E6974200A24F9490025F9564000646E616D6506F95CB007F958B005F94900

	cborPld := msg
	cborPld.Payload = cborBytes

	tooManyMsg := msg
	tooManyMsg.Payload = tooManyBytes

	val := 52.0
	sum := 110.0
	msgs := []senml.Message{
		{
			Subtopic:   "subtopic",
			Publisher:  "publisher",
			Protocol:   "protocol",
			Name:       "base-namename",
			Unit:       "unit",
			Time:       400,
			UpdateTime: 150,
			Value:      &val,
			Sum:        &sum,
		},
	}

	cases := []struct {
		desc string
		msg  protomfx.Message
		msgs interface{}
		err  error
	}{
		{
			desc: "test normalize CBOR",
			msg:  cborPld,
			msgs: msgs,
			err:  nil,
		},
		{
			desc: "test invalid payload",
			msg:  tooManyMsg,
			msgs: nil,
			err:  mfsenml.ErrTooManyValues,
		},
	}

	for _, tc := range cases {
		msgs, err := tr.Transform(tc.msg)
		assert.Equal(t, tc.msgs, msgs, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.msgs, msgs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}
