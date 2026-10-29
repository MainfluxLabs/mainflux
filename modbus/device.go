package modbus

import (
	"encoding/binary"
	"fmt"
	"strings"

	gbmodbus "github.com/goburrow/modbus"
)

const (
	errReadRegisters  = "failed to read registers"
	errWriteRegisters = "failed to write registers"
)

func isWriteFunc(funcCode string) bool {
	switch funcCode {
	case WriteCoilsFunc, WriteHoldingRegistersFunc:
		return true
	default:
		return false
	}
}

// readFunc returns the read function code for the register table that a write function code targets.
func readFunc(funcCode string) string {
	switch funcCode {
	case WriteCoilsFunc:
		return ReadCoilsFunc
	case WriteHoldingRegistersFunc:
		return ReadHoldingRegistersFunc
	default:
		return funcCode
	}
}

func readData(mc gbmodbus.Client, client Client, blocks []Block) (map[string][]byte, error) {
	data := make(map[string][]byte)

	for _, block := range blocks {
		fields := blockFields(client.DataFields, block)

		raw, err := readBlock(mc, client.FunctionCode, block.Start, block.Length)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: read %s starting at %d, length %d, fields [%s]: %w",
				errReadRegisters, client.FunctionCode, block.Start, block.Length,
				fieldNames(fields), err,
			)
		}

		for _, field := range fields {
			bytes, err := extractFieldBytes(raw, field, block, client.FunctionCode)
			if err != nil {
				return nil, fmt.Errorf("field %s@%d: %w", field.Name, field.Address, err)
			}
			data[field.Name] = bytes
		}
	}
	return data, nil
}

func readBlock(mc gbmodbus.Client, funcCode string, start, length uint16) ([]byte, error) {
	switch funcCode {
	case ReadCoilsFunc:
		return mc.ReadCoils(start, length)
	case ReadDiscreteInputsFunc:
		return mc.ReadDiscreteInputs(start, length)
	case ReadInputRegistersFunc:
		return mc.ReadInputRegisters(start, length)
	default:
		return mc.ReadHoldingRegisters(start, length)
	}
}

func writeData(mc gbmodbus.Client, client Client, blocks []Block) error {
	for _, block := range blocks {
		fields := blockFields(client.DataFields, block)

		// Encoding errors are returned unwrapped to preserve the ErrInvalidWriteValue and ErrValueOutOfRange sentinels.
		raw, err := encodeBlock(client.FunctionCode, block, fields)
		if err != nil {
			return err
		}

		if err := writeBlock(mc, client.FunctionCode, block, raw); err != nil {
			return fmt.Errorf(
				"%s: write %s starting at %d, length %d, fields [%s]: %w",
				errWriteRegisters, client.FunctionCode, block.Start, block.Length,
				fieldNames(fields), err,
			)
		}
	}

	return nil
}

// writeBlock uses function codes 0x05 and 0x06 for single-element blocks,
// because not all devices implement 0x0F and 0x10.
func writeBlock(mc gbmodbus.Client, funcCode string, block Block, raw []byte) error {
	switch funcCode {
	case WriteCoilsFunc:
		if block.Length == 1 {
			coilValue := uint16(0x0000)
			if raw[0] != 0 {
				coilValue = 0xFF00
			}

			_, err := mc.WriteSingleCoil(block.Start, coilValue)
			return err
		}

		_, err := mc.WriteMultipleCoils(block.Start, block.Length, raw)
		return err
	default:
		if block.Length == 1 {
			_, err := mc.WriteSingleRegister(block.Start, binary.BigEndian.Uint16(raw))
			return err
		}

		_, err := mc.WriteMultipleRegisters(block.Start, block.Length, raw)
		return err
	}
}

// fieldNames formats fields as name@address pairs for error messages.
func fieldNames(fields []DataField) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, fmt.Sprintf("%s@%d", field.Name, field.Address))
	}

	return strings.Join(names, ", ")
}
