// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package modbus

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	gbmodbus "github.com/goburrow/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modbusClientMock records the last write call made through it, so tests
// can assert on address/value/quantity without a real Modbus device.
type modbusClientMock struct {
	gbmodbus.Client
	lastAddress  uint16
	lastValue    uint16
	lastQuantity uint16
	lastBytes    []byte
	err          error
}

func (f *modbusClientMock) WriteSingleCoil(address, value uint16) ([]byte, error) {
	f.lastAddress, f.lastValue = address, value
	return nil, f.err
}

func (f *modbusClientMock) WriteSingleRegister(address, value uint16) ([]byte, error) {
	f.lastAddress, f.lastValue = address, value
	return nil, f.err
}

func (f *modbusClientMock) WriteMultipleCoils(address, quantity uint16, value []byte) ([]byte, error) {
	f.lastAddress, f.lastQuantity, f.lastBytes = address, quantity, value
	return nil, f.err
}

func (f *modbusClientMock) WriteMultipleRegisters(address, quantity uint16, value []byte) ([]byte, error) {
	f.lastAddress, f.lastQuantity, f.lastBytes = address, quantity, value
	return nil, f.err
}

func TestReorderBytes(t *testing.T) {
	cases := []struct {
		desc  string
		input []byte
		order string
		want  []byte
	}{
		{
			desc:  "ABCD (big-endian) returns same order",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			order: ByteOrderABCD,
			want:  []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			desc:  "DCBA (little-endian) reverses all bytes",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			order: ByteOrderDCBA,
			want:  []byte{0x04, 0x03, 0x02, 0x01},
		},
		{
			desc:  "CDAB (mid-big) swaps word pairs",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			order: ByteOrderCDAB,
			want:  []byte{0x03, 0x04, 0x01, 0x02},
		},
		{
			desc:  "BADC (byte-swap) swaps bytes within each word",
			input: []byte{0x01, 0x02, 0x03, 0x04},
			order: ByteOrderBADC,
			want:  []byte{0x02, 0x01, 0x04, 0x03},
		},
		{
			desc:  "DCBA with 2-byte input reverses",
			input: []byte{0x01, 0x02},
			order: ByteOrderDCBA,
			want:  []byte{0x02, 0x01},
		},
		{
			desc:  "CDAB with 2-byte input is unchanged (guard: only applies to 4 bytes)",
			input: []byte{0x01, 0x02},
			order: ByteOrderCDAB,
			want:  []byte{0x01, 0x02},
		},
		{
			desc:  "unknown order returns original",
			input: []byte{0x01, 0x02},
			order: "UNKNOWN",
			want:  []byte{0x01, 0x02},
		},
		{
			desc:  "does not modify original slice",
			input: []byte{0xAA, 0xBB, 0xCC, 0xDD},
			order: ByteOrderDCBA,
			want:  []byte{0xDD, 0xCC, 0xBB, 0xAA},
		},
	}

	for _, tc := range cases {
		original := make([]byte, len(tc.input))
		copy(original, tc.input)

		got := reorderBytes(tc.input, tc.order)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
		// verify original is unchanged
		assert.Equal(t, original, tc.input, fmt.Sprintf("%s: original should not be modified", tc.desc))
	}
}

func TestExtractFieldBytesRegisters(t *testing.T) {
	// 4 registers = 8 bytes: [0x00,0x01, 0x00,0x02, 0x00,0x03, 0x00,0x04]
	raw := []byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04}
	block := Block{Start: 0, Length: 4}

	cases := []struct {
		desc     string
		field    DataField
		block    Block
		funcCode string
		want     []byte
		wantErr  bool
	}{
		{
			desc:     "extract first register (address 0, length 1)",
			field:    DataField{Name: "f1", Address: 0, Length: 1},
			block:    block,
			funcCode: ReadHoldingRegistersFunc,
			want:     []byte{0x00, 0x01},
			wantErr:  false,
		},
		{
			desc:     "extract second register (address 1, length 1)",
			field:    DataField{Name: "f2", Address: 1, Length: 1},
			block:    block,
			funcCode: ReadHoldingRegistersFunc,
			want:     []byte{0x00, 0x02},
			wantErr:  false,
		},
		{
			desc:     "extract two registers starting at address 2",
			field:    DataField{Name: "f3", Address: 2, Length: 2},
			block:    block,
			funcCode: ReadInputRegistersFunc,
			want:     []byte{0x00, 0x03, 0x00, 0x04},
			wantErr:  false,
		},
		{
			desc:     "out of range returns error",
			field:    DataField{Name: "f4", Address: 0, Length: 10},
			block:    block,
			funcCode: ReadHoldingRegistersFunc,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		got, err := extractFieldBytes(raw, tc.field, tc.block, tc.funcCode)
		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestExtractFieldBytesCoils(t *testing.T) {
	// byte 0: 0b00001011 → bits 0,1,3 are ON
	raw := []byte{0x0B}
	block := Block{Start: 0, Length: 8}

	cases := []struct {
		desc    string
		field   DataField
		want    []byte
		wantErr bool
	}{
		{
			desc:    "coil at bit 0 is ON",
			field:   DataField{Name: "coil0", Address: 0},
			want:    []byte{0x01},
			wantErr: false,
		},
		{
			desc:    "coil at bit 1 is ON",
			field:   DataField{Name: "coil1", Address: 1},
			want:    []byte{0x01},
			wantErr: false,
		},
		{
			desc:    "coil at bit 2 is OFF",
			field:   DataField{Name: "coil2", Address: 2},
			want:    []byte{0x00},
			wantErr: false,
		},
		{
			desc:    "coil at bit 3 is ON",
			field:   DataField{Name: "coil3", Address: 3},
			want:    []byte{0x01},
			wantErr: false,
		},
		{
			desc:    "out of range coil returns error",
			field:   DataField{Name: "coilX", Address: 8},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		got, err := extractFieldBytes(raw, tc.field, block, ReadCoilsFunc)
		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

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

func TestReadFunc(t *testing.T) {
	cases := []struct {
		desc     string
		funcCode string
		want     string
	}{
		{
			desc:     "WriteCoils returns ReadCoils",
			funcCode: WriteCoilsFunc,
			want:     ReadCoilsFunc,
		},
		{
			desc:     "WriteHoldingRegisters returns ReadHoldingRegisters",
			funcCode: WriteHoldingRegistersFunc,
			want:     ReadHoldingRegistersFunc,
		},
		{
			desc:     "read function code is returned unchanged",
			funcCode: ReadInputRegistersFunc,
			want:     ReadInputRegistersFunc,
		},
	}

	for _, tc := range cases {
		got := readFunc(tc.funcCode)
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

func TestCreateEntry(t *testing.T) {
	cases := []struct {
		desc  string
		value any
		unit  string
		want  map[string]any
	}{
		{
			desc:  "entry with unit",
			value: float64(25.5),
			unit:  "Celsius",
			want:  map[string]any{"value": float64(25.5), "unit": "Celsius"},
		},
		{
			desc:  "entry without unit",
			value: int16(100),
			unit:  "",
			want:  map[string]any{"value": int16(100)},
		},
		{
			desc:  "entry with bool value",
			value: true,
			unit:  "",
			want:  map[string]any{"value": true},
		},
	}

	for _, tc := range cases {
		got := createEntry(tc.value, tc.unit)
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestReadNumericField(t *testing.T) {
	cases := []struct {
		desc    string
		data    []byte
		scale   float64
		typ     string
		want    any
		wantErr bool
	}{
		{
			desc:    "int16 positive value",
			data:    []byte{0x00, 0x64},
			scale:   0,
			typ:     Int16Type,
			want:    int16(100),
			wantErr: false,
		},
		{
			desc:    "int16 negative value",
			data:    []byte{0xFF, 0x9C},
			scale:   0,
			typ:     Int16Type,
			want:    int16(-100),
			wantErr: false,
		},
		{
			desc:    "uint16 value",
			data:    []byte{0x01, 0x00},
			scale:   0,
			typ:     Uint16Type,
			want:    uint16(256),
			wantErr: false,
		},
		{
			desc:    "int32 value",
			data:    []byte{0x00, 0x01, 0x86, 0xA0},
			scale:   0,
			typ:     Int32Type,
			want:    int32(100000),
			wantErr: false,
		},
		{
			desc:    "uint32 value",
			data:    []byte{0x00, 0x01, 0x86, 0xA0},
			scale:   0,
			typ:     Uint32Type,
			want:    uint32(100000),
			wantErr: false,
		},
		{
			desc:    "float32 value",
			data:    float32ToBytes(3.14),
			scale:   0,
			typ:     Float32Type,
			want:    float32(3.14),
			wantErr: false,
		},
		{
			desc:    "int16 with scale applied",
			data:    []byte{0x00, 0x0A},
			scale:   0.1,
			typ:     Int16Type,
			want:    float64(10) * 0.1,
			wantErr: false,
		},
		{
			desc:    "int16 insufficient bytes",
			data:    []byte{0x01},
			scale:   0,
			typ:     Int16Type,
			want:    nil,
			wantErr: true,
		},
		{
			desc:    "int32 insufficient bytes",
			data:    []byte{0x00, 0x01},
			scale:   0,
			typ:     Int32Type,
			want:    nil,
			wantErr: true,
		},
		{
			desc:    "float32 insufficient bytes",
			data:    []byte{0x00, 0x01},
			scale:   0,
			typ:     Float32Type,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		var (
			got any
			err error
		)

		switch tc.typ {
		case Int16Type:
			got, err = readNumericField[int16](tc.data, tc.scale)
		case Uint16Type:
			got, err = readNumericField[uint16](tc.data, tc.scale)
		case Int32Type:
			got, err = readNumericField[int32](tc.data, tc.scale)
		case Uint32Type:
			got, err = readNumericField[uint32](tc.data, tc.scale)
		case Float32Type:
			got, err = readNumericField[float32](tc.data, tc.scale)
		}

		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		if tc.typ == Float32Type && tc.scale == 0 {
			// compare float32 with tolerance
			gotF, ok := got.(float32)
			require.True(t, ok)
			assert.InDelta(t, tc.want.(float32), gotF, 0.001, fmt.Sprintf("%s: unexpected result", tc.desc))
		} else {
			assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
		}
	}
}

func TestFormatRegistersPayload(t *testing.T) {
	cases := []struct {
		desc    string
		data    map[string][]byte
		fields  []DataField
		check   func(t *testing.T, got map[string]any)
		wantErr bool
	}{
		{
			desc: "int16 field",
			data: map[string][]byte{"temp": {0x00, 0x1E}},
			fields: []DataField{
				{Name: "temp", Type: Int16Type, ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				// JSON unmarshal converts all numbers to float64
				entry := got["temp"].(map[string]any)
				assert.InDelta(t, float64(30), entry["value"].(float64), 0.001)
			},
		},
		{
			desc: "float32 field with unit",
			data: map[string][]byte{"pressure": float32ToBytes(1.5)},
			fields: []DataField{
				{Name: "pressure", Type: Float32Type, Unit: "bar", ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				// JSON unmarshal converts float32 to float64
				entry := got["pressure"].(map[string]any)
				assert.InDelta(t, 1.5, entry["value"].(float64), 0.01)
				assert.Equal(t, "bar", entry["unit"])
			},
		},
		{
			desc: "bool field - value 1 is true",
			data: map[string][]byte{"flag": {0x00, 0x01}},
			fields: []DataField{
				{Name: "flag", Type: BoolType, ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				entry := got["flag"].(map[string]any)
				assert.Equal(t, true, entry["value"])
			},
		},
		{
			desc: "string field",
			data: map[string][]byte{"label": []byte("AB\x00")},
			fields: []DataField{
				{Name: "label", Type: StringType, ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				entry := got["label"].(map[string]any)
				assert.Equal(t, "AB", entry["value"])
			},
		},
		{
			desc: "field with scale",
			data: map[string][]byte{"voltage": {0x00, 0x64}},
			fields: []DataField{
				{Name: "voltage", Type: Int16Type, Scale: 0.1, ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				entry := got["voltage"].(map[string]any)
				assert.InDelta(t, 10.0, entry["value"].(float64), 0.001)
			},
		},
		{
			desc: "missing field in data is skipped",
			data: map[string][]byte{},
			fields: []DataField{
				{Name: "absent", Type: Int16Type, ByteOrder: ByteOrderABCD},
			},
			check: func(t *testing.T, got map[string]any) {
				_, exists := got["absent"]
				assert.False(t, exists)
			},
		},
		{
			desc: "bool field with too few bytes returns error",
			data: map[string][]byte{"flag": {0x01}},
			fields: []DataField{
				{Name: "flag", Type: BoolType, ByteOrder: ByteOrderABCD},
			},
			wantErr: true,
		},
		{
			desc: "float32 field with too few bytes returns error",
			data: map[string][]byte{"temp": {0x00, 0x01}},
			fields: []DataField{
				{Name: "temp", Type: Float32Type, ByteOrder: ByteOrderABCD},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		raw, err := formatPayload(tc.data, tc.fields, ReadHoldingRegistersFunc)
		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		var result map[string]any
		require.Nil(t, json.Unmarshal(raw, &result), "should produce valid JSON")
		tc.check(t, result)
	}
}

func TestFormatCoilsPayload(t *testing.T) {
	cases := []struct {
		desc   string
		data   map[string][]byte
		fields []DataField
		want   map[string]any
	}{
		{
			desc:   "coil ON (0x01)",
			data:   map[string][]byte{"relay": {0x01}},
			fields: []DataField{{Name: "relay"}},
			want:   map[string]any{"relay": true},
		},
		{
			desc:   "coil OFF (0x00)",
			data:   map[string][]byte{"valve": {0x00}},
			fields: []DataField{{Name: "valve"}},
			want:   map[string]any{"valve": false},
		},
		{
			desc:   "missing coil data is skipped",
			data:   map[string][]byte{},
			fields: []DataField{{Name: "absent"}},
			want:   map[string]any{},
		},
	}

	for _, tc := range cases {
		raw, err := formatPayload(tc.data, tc.fields, ReadCoilsFunc)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		var result map[string]any
		require.Nil(t, json.Unmarshal(raw, &result))
		assert.Equal(t, tc.want, result, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestEncodeNumericField(t *testing.T) {
	cases := []struct {
		desc    string
		value   float64
		typ     string
		scale   float64
		want    []byte
		wantErr bool
	}{
		{desc: "int16 positive", value: 100, typ: Int16Type, want: []byte{0x00, 0x64}},
		{desc: "int16 negative", value: -100, typ: Int16Type, want: []byte{0xFF, 0x9C}},
		{desc: "uint16", value: 256, typ: Uint16Type, want: []byte{0x01, 0x00}},
		{desc: "int32", value: 100000, typ: Int32Type, want: []byte{0x00, 0x01, 0x86, 0xA0}},
		{desc: "uint32", value: 100000, typ: Uint32Type, want: []byte{0x00, 0x01, 0x86, 0xA0}},
		{desc: "float32", value: 1.5, typ: Float32Type, want: float32ToBytes(1.5)},
		{desc: "int16 with scale divides value before encoding", value: 10.0, typ: Int16Type, scale: 0.1, want: []byte{0x00, 0x64}},
		{desc: "unsupported type returns error", value: 1, typ: StringType, wantErr: true},
	}

	for _, tc := range cases {
		got, err := encodeNumericField(tc.value, tc.typ, tc.scale)
		if tc.wantErr {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.want, got, fmt.Sprintf("%s: unexpected result", tc.desc))
	}
}

func TestValidateWriteValue(t *testing.T) {
	cases := []struct {
		desc  string
		value float64
		typ   string
		scale float64
		err   error
	}{
		{
			desc:  "validate int16 value within range",
			value: 100,
			typ:   Int16Type,
			err:   nil,
		},
		{
			desc:  "validate int16 value out of range",
			value: 100000,
			typ:   Int16Type,
			err:   ErrValueOutOfRange,
		},
		{
			desc:  "validate uint16 negative value",
			value: -1,
			typ:   Uint16Type,
			err:   ErrValueOutOfRange,
		},
		{
			desc:  "validate int16 value out of range after scale",
			value: 10,
			typ:   Int16Type,
			scale: 0.0001,
			err:   ErrValueOutOfRange,
		},
		{
			desc:  "validate float32 value",
			value: 21.5,
			typ:   Float32Type,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := ValidateWriteValue(tc.value, tc.typ, tc.scale)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestWriteData(t *testing.T) {
	cases := []struct {
		desc     string
		funcCode string
		fields   []DataField
		address  uint16
		value    uint16
		quantity uint16
		bytes    []byte
		err      error
	}{
		{
			desc:     "write single coil ON",
			funcCode: WriteCoilsFunc,
			fields: []DataField{
				{Name: "pump", Type: BoolType, Address: 5, Length: 1, Value: true},
			},
			address: 5,
			value:   0xFF00,
			err:     nil,
		},
		{
			desc:     "write single coil OFF",
			funcCode: WriteCoilsFunc,
			fields: []DataField{
				{Name: "pump", Type: BoolType, Address: 5, Length: 1, Value: false},
			},
			address: 5,
			value:   0x0000,
			err:     nil,
		},
		{
			desc:     "write adjacent coils packed LSB-first",
			funcCode: WriteCoilsFunc,
			fields: []DataField{
				{Name: "c0", Type: BoolType, Address: 0, Length: 1, Value: true},
				{Name: "c1", Type: BoolType, Address: 1, Length: 1, Value: false},
				{Name: "c2", Type: BoolType, Address: 2, Length: 1, Value: true},
			},
			address:  0,
			quantity: 3,
			bytes:    []byte{0x05},
			err:      nil,
		},
		{
			desc:     "write coil with non-bool value",
			funcCode: WriteCoilsFunc,
			fields: []DataField{
				{Name: "pump", Type: BoolType, Address: 0, Length: 1, Value: float64(1)},
			},
			err: ErrInvalidWriteValue,
		},
		{
			desc:     "write single uint16 register",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "mode", Type: Uint16Type, Address: 100, Length: 1, Value: float64(256)},
			},
			address: 100,
			value:   256,
			err:     nil,
		},
		{
			// DCBA order reverses 0x0100 into 0x0001.
			desc:     "write single uint16 register with byte order",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "mode", Type: Uint16Type, ByteOrder: ByteOrderDCBA, Address: 100, Length: 1, Value: float64(256)},
			},
			address: 100,
			value:   1,
			err:     nil,
		},
		{
			desc:     "write single float32 into two registers with byte order",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "setpoint", Type: Float32Type, ByteOrder: ByteOrderDCBA, Address: 200, Length: 2, Value: 1.5},
			},
			address:  200,
			quantity: 2,
			bytes:    reorderBytes(float32ToBytes(1.5), ByteOrderDCBA),
			err:      nil,
		},
		{
			desc:     "write adjacent registers with byte order",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "setpoint", Type: Float32Type, ByteOrder: ByteOrderDCBA, Address: 11, Length: 2, Value: 1.5},
				{Name: "enabled", Type: BoolType, Address: 10, Length: 1, Value: true},
			},
			address:  10,
			quantity: 3,
			bytes:    append([]byte{0x00, 0x01}, reorderBytes(float32ToBytes(1.5), ByteOrderDCBA)...),
			err:      nil,
		},
		{
			// DCBA order reverses 0x0001 into 0x0100.
			desc:     "write single bool register with byte order",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "enabled", Type: BoolType, ByteOrder: ByteOrderDCBA, Address: 10, Length: 1, Value: true},
			},
			address: 10,
			value:   0x0100,
			err:     nil,
		},
		{
			desc:     "write register with non-numeric value",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "mode", Type: Uint16Type, Address: 0, Length: 1, Value: "256"},
			},
			err: ErrInvalidWriteValue,
		},
		{
			desc:     "write register with unsupported type",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "label", Type: StringType, Address: 0, Length: 2, Value: float64(1)},
			},
			err: errors.New(fmt.Sprintf("%s: %s", errUnsupportedWriteType, StringType)),
		},
		{
			desc:     "write register with value out of range",
			funcCode: WriteHoldingRegistersFunc,
			fields: []DataField{
				{Name: "mode", Type: Int16Type, Address: 0, Length: 1, Value: float64(100000)},
			},
			err: ErrValueOutOfRange,
		},
	}

	for _, tc := range cases {
		mc := &modbusClientMock{}
		client := Client{FunctionCode: tc.funcCode, DataFields: tc.fields}
		err := writeData(mc, client, createWriteBlocks(client.DataFields, getBlockMaxLen(tc.funcCode)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.address, mc.lastAddress, fmt.Sprintf("%s: unexpected address", tc.desc))
		assert.Equal(t, tc.value, mc.lastValue, fmt.Sprintf("%s: unexpected value", tc.desc))
		assert.Equal(t, tc.quantity, mc.lastQuantity, fmt.Sprintf("%s: unexpected quantity", tc.desc))
		assert.Equal(t, tc.bytes, mc.lastBytes, fmt.Sprintf("%s: unexpected bytes", tc.desc))
	}
}

// float32ToBytes converts a float32 to its big-endian byte representation.
func float32ToBytes(f float32) []byte {
	bits := math.Float32bits(f)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, bits)
	return b
}
