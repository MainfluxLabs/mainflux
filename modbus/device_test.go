// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package modbus

import (
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	gbmodbus "github.com/goburrow/modbus"
	"github.com/stretchr/testify/assert"
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
			err: ErrUnsupportedWriteType,
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
