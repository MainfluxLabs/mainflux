// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package modbus

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCreateBlocks(t *testing.T) {
	cases := []struct {
		desc   string
		fields []DataField
		maxLen int
		want   []Block
	}{
		{
			desc:   "empty fields returns nil",
			fields: []DataField{},
			maxLen: 125,
			want:   nil,
		},
		{
			desc: "single field creates one block",
			fields: []DataField{
				{Address: 0, Length: 1},
			},
			maxLen: 125,
			want:   []Block{{Start: 0, Length: 1}},
		},
		{
			desc: "contiguous fields merge into one block",
			fields: []DataField{
				{Address: 0, Length: 1},
				{Address: 1, Length: 1},
				{Address: 2, Length: 1},
			},
			maxLen: 125,
			want:   []Block{{Start: 0, Length: 3}},
		},
		{
			desc: "fields exceeding maxLen split into separate blocks",
			fields: []DataField{
				{Address: 0, Length: 1},
				{Address: 200, Length: 1},
			},
			maxLen: 125,
			want: []Block{
				{Start: 0, Length: 1},
				{Start: 200, Length: 1},
			},
		},
		{
			desc: "fields are sorted by address before grouping",
			fields: []DataField{
				{Address: 5, Length: 1},
				{Address: 0, Length: 1},
				{Address: 3, Length: 1},
			},
			maxLen: 125,
			want:   []Block{{Start: 0, Length: 6}},
		},
	}

	for _, tc := range cases {
		got := createBlocks(tc.fields, tc.maxLen)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestCreateWriteBlocks(t *testing.T) {
	cases := []struct {
		desc   string
		fields []DataField
		maxLen int
		want   []Block
	}{
		{
			desc:   "empty fields returns nil",
			fields: []DataField{},
			maxLen: maxRegs,
			want:   nil,
		},
		{
			desc: "adjacent fields merge into one block",
			fields: []DataField{
				{Address: 2, Length: 2},
				{Address: 0, Length: 1},
				{Address: 1, Length: 1},
			},
			maxLen: maxRegs,
			want:   []Block{{Start: 0, Length: 4}},
		},
		{
			desc: "fields separated by a gap split into separate blocks",
			fields: []DataField{
				{Address: 0, Length: 1},
				{Address: 2, Length: 1},
			},
			maxLen: maxRegs,
			want: []Block{
				{Start: 0, Length: 1},
				{Start: 2, Length: 1},
			},
		},
		{
			desc: "adjacent fields exceeding maxLen split into separate blocks",
			fields: []DataField{
				{Address: 0, Length: 2},
				{Address: 2, Length: 2},
			},
			maxLen: 3,
			want: []Block{
				{Start: 0, Length: 2},
				{Start: 2, Length: 2},
			},
		},
	}

	for _, tc := range cases {
		got := createWriteBlocks(tc.fields, tc.maxLen)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestGetBlockMaxLen(t *testing.T) {
	cases := []struct {
		desc     string
		funcCode string
		want     int
	}{
		{
			desc:     "ReadCoils returns maxBits",
			funcCode: ReadCoilsFunc,
			want:     maxBits,
		},
		{
			desc:     "ReadDiscreteInputs returns maxBits",
			funcCode: ReadDiscreteInputsFunc,
			want:     maxBits,
		},
		{
			desc:     "ReadHoldingRegisters returns maxRegs",
			funcCode: ReadHoldingRegistersFunc,
			want:     maxRegs,
		},
		{
			desc:     "ReadInputRegisters returns maxRegs",
			funcCode: ReadInputRegistersFunc,
			want:     maxRegs,
		},
		{
			desc:     "WriteCoils returns maxWriteBits",
			funcCode: WriteCoilsFunc,
			want:     maxWriteBits,
		},
		{
			desc:     "WriteHoldingRegisters returns maxRegs",
			funcCode: WriteHoldingRegistersFunc,
			want:     maxRegs,
		},
		{
			desc:     "unknown function code returns maxRegs",
			funcCode: "unknown",
			want:     maxRegs,
		},
	}

	for _, tc := range cases {
		got := getBlockMaxLen(tc.funcCode)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestCalcFieldLengths(t *testing.T) {
	cases := []struct {
		desc   string
		fields []DataField
		want   []DataField
	}{
		{
			desc:   "int16 gets length 1",
			fields: []DataField{{Type: Int16Type}},
			want:   []DataField{{Type: Int16Type, Length: 1}},
		},
		{
			desc:   "uint16 gets length 1",
			fields: []DataField{{Type: Uint16Type}},
			want:   []DataField{{Type: Uint16Type, Length: 1}},
		},
		{
			desc:   "bool gets length 1",
			fields: []DataField{{Type: BoolType}},
			want:   []DataField{{Type: BoolType, Length: 1}},
		},
		{
			desc:   "int32 gets length 2",
			fields: []DataField{{Type: Int32Type}},
			want:   []DataField{{Type: Int32Type, Length: 2}},
		},
		{
			desc:   "uint32 gets length 2",
			fields: []DataField{{Type: Uint32Type}},
			want:   []DataField{{Type: Uint32Type, Length: 2}},
		},
		{
			desc:   "float32 gets length 2",
			fields: []DataField{{Type: Float32Type}},
			want:   []DataField{{Type: Float32Type, Length: 2}},
		},
		{
			desc:   "string length is unchanged",
			fields: []DataField{{Type: StringType, Length: 5}},
			want:   []DataField{{Type: StringType, Length: 5}},
		},
		{
			desc: "mixed types get correct lengths",
			fields: []DataField{
				{Type: Int16Type},
				{Type: Float32Type},
				{Type: StringType, Length: 3},
			},
			want: []DataField{
				{Type: Int16Type, Length: 1},
				{Type: Float32Type, Length: 2},
				{Type: StringType, Length: 3},
			},
		},
	}

	for _, tc := range cases {
		got := calcFieldLengths(tc.fields)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestValidateFieldAddresses(t *testing.T) {
	cases := []struct {
		desc   string
		fields []DataField
		err    error
	}{
		{
			desc: "validate field ending at the last address",
			fields: []DataField{
				{Name: "last", Address: 65534, Length: 2},
			},
			err: nil,
		},
		{
			desc: "validate field ending past the last address",
			fields: []DataField{
				{Name: "wrapped", Address: 65535, Length: 2},
			},
			err: ErrAddressOutOfRange,
		},
	}

	for _, tc := range cases {
		err := validateFieldAddresses(tc.fields)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)
	}
}
