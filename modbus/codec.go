package modbus

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// errNotEnoughBytes indicates a device response holds fewer bytes than the requested fields need.
var errNotEnoughBytes = errors.New("not enough bytes to read")

// encodeBlock converts the field values of a block into its write request payload.
func encodeBlock(funcCode string, block Block, fields []DataField) ([]byte, error) {
	switch funcCode {
	case WriteCoilsFunc:
		// Modbus packs coils LSB-first.
		bits := make([]byte, (block.Length+7)/8)
		for _, field := range fields {
			v, ok := field.Value.(bool)
			if !ok {
				return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected bool value for field %s", field.Name))
			}
			if v {
				offset := int(field.Address - block.Start)
				byteIndex := offset / 8
				bitIndex := uint(offset % 8)
				bits[byteIndex] |= 1 << bitIndex
			}
		}
		return bits, nil
	default:
		raw := make([]byte, int(block.Length)*2)
		for _, field := range fields {
			b, err := encodeRegisterField(field)
			if err != nil {
				return nil, err
			}
			copy(raw[int(field.Address-block.Start)*2:], b)
		}
		return raw, nil
	}
}

// encodeRegisterField converts the field value into raw register bytes in the field byte order.
func encodeRegisterField(field DataField) ([]byte, error) {
	var raw []byte
	switch field.Type {
	case BoolType:
		v, ok := field.Value.(bool)
		if !ok {
			return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected bool value for field %s", field.Name))
		}

		raw = []byte{0x00, 0x00}
		if v {
			raw[1] = 0x01
		}
	default:
		v, ok := field.Value.(float64)
		if !ok {
			return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected numeric value for field %s", field.Name))
		}

		var err error
		raw, err = encodeNumericField(v, field.Type, field.Scale)
		if err != nil {
			return nil, err
		}
	}

	return reorderBytes(raw, field.ByteOrder), nil
}

// ValidateWriteValue checks that value, after scale is applied, fits the numeric register type.
func ValidateWriteValue(value float64, typ string, scale float64) error {
	_, err := encodeNumericField(value, typ, scale)
	return err
}

// encodeNumericField converts an engineering-unit value back into raw big-endian
// device bytes, undoing the scale applied by readNumericField.
func encodeNumericField(value float64, typ string, scale float64) ([]byte, error) {
	if scale != 0 {
		value /= scale
	}

	switch typ {
	case Int16Type:
		rounded := math.Round(value)
		if rounded < math.MinInt16 || rounded > math.MaxInt16 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(int16(rounded)))
		return b, nil
	case Uint16Type:
		rounded := math.Round(value)
		if rounded < 0 || rounded > math.MaxUint16 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(rounded))
		return b, nil
	case Int32Type:
		rounded := math.Round(value)
		if rounded < math.MinInt32 || rounded > math.MaxInt32 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(int32(rounded)))
		return b, nil
	case Uint32Type:
		rounded := math.Round(value)
		if rounded < 0 || rounded > math.MaxUint32 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(rounded))
		return b, nil
	case Float32Type:
		if math.Abs(value) > math.MaxFloat32 {
			err := fmt.Errorf("%v out of range for %s", value, typ)
			return nil, errors.Wrap(ErrValueOutOfRange, err)
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, math.Float32bits(float32(value)))
		return b, nil
	default:
		return nil, errors.Wrap(ErrUnsupportedWriteType, fmt.Errorf("type %s is not writable", typ))
	}
}

func readNumericField[T int16 | uint16 | int32 | uint32 | float32](data []byte, scale float64) (any, error) {
	var value T
	switch any(value).(type) {
	case int16:
		if len(data) < 2 {
			err := fmt.Errorf("expected 2 bytes for int16, got %d", len(data))
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}
		value = T(int16(binary.BigEndian.Uint16(data)))
	case uint16:
		if len(data) < 2 {
			err := fmt.Errorf("expected 2 bytes for uint16, got %d", len(data))
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}
		value = T(binary.BigEndian.Uint16(data))
	case int32:
		if len(data) < 4 {
			err := fmt.Errorf("expected 4 bytes for int32, got %d", len(data))
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}
		value = T(int32(binary.BigEndian.Uint32(data)))
	case uint32:
		if len(data) < 4 {
			err := fmt.Errorf("expected 4 bytes for uint32, got %d", len(data))
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}
		value = T(binary.BigEndian.Uint32(data))
	case float32:
		if len(data) < 4 {
			err := fmt.Errorf("expected 4 bytes for float32, got %d", len(data))
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}
		bits := binary.BigEndian.Uint32(data)
		value = T(math.Float32frombits(bits))
	}

	if scale != 0 {
		return float64(value) * scale, nil
	}
	return value, nil
}

func reorderBytes(data []byte, order string) []byte {
	bytes := append([]byte(nil), data...)

	switch order {
	case ByteOrderABCD:
		return bytes
	case ByteOrderDCBA:
		slices.Reverse(bytes)
	case ByteOrderCDAB:
		if len(bytes) == 4 {
			bytes = []byte{bytes[2], bytes[3], bytes[0], bytes[1]}
		}
	case ByteOrderBADC:
		if len(bytes) == 4 {
			bytes = []byte{bytes[1], bytes[0], bytes[3], bytes[2]}
		}
	}
	return bytes
}

func extractFieldBytes(raw []byte, field DataField, block Block, funcCode string) ([]byte, error) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		// coils/discrete inputs: single bits (1 bit = 1 value)
		// calculate bit offset relative to block start
		bitOffset := int(field.Address - block.Start)

		// determine byte and bit inside byte that holds value
		byteIndex := bitOffset / 8
		bitIndex := uint(bitOffset % 8)

		if byteIndex >= len(raw) {
			err := fmt.Errorf(
				"field %s at address %d: out of range for coil block starting at %d, length %d",
				field.Name, field.Address, block.Start, block.Length,
			)
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}

		// extract bit: mask byte and check if bit is set
		bit := (raw[byteIndex] & (1 << bitIndex)) != 0

		// convert bit (bool) to single byte (0x01 = ON, 0x00 = OFF)
		if bit {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil
	default:
		// registers: each is 2 bytes (16-bit)
		// convert register offset to byte offset
		startByte := int(field.Address-block.Start) * 2
		lenByte := int(field.Length) * 2

		if startByte+lenByte > len(raw) {
			err := fmt.Errorf(
				"field %s at address %d, length %d: out of range for register block "+
					"starting at %d, length %d",
				field.Name, field.Address, field.Length, block.Start, block.Length,
			)
			return nil, errors.Wrap(errNotEnoughBytes, err)
		}

		return raw[startByte : startByte+lenByte], nil
	}
}

func formatPayload(data map[string][]byte, fields []DataField, funcCode string) ([]byte, error) {
	result, err := buildFieldValues(data, fields, funcCode)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func buildFieldValues(data map[string][]byte, fields []DataField, funcCode string) (map[string]any, error) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		return buildCoilValues(data, fields)
	default:
		return buildRegisterValues(data, fields)
	}
}

func buildRegisterValues(data map[string][]byte, fields []DataField) (map[string]any, error) {
	result := make(map[string]any)

	for _, f := range fields {
		raw, ok := data[f.Name]
		if !ok {
			continue
		}

		raw = reorderBytes(raw, f.ByteOrder)

		var (
			value any
			err   error
		)

		switch f.Type {
		case BoolType:
			if len(raw) < 2 {
				err = fmt.Errorf("expected 2 bytes for bool, got %d", len(raw))
				return nil, errors.Wrap(errNotEnoughBytes, err)
			}
			bv := binary.BigEndian.Uint16(raw)
			value = bv == 1
		case Int16Type:
			value, err = readNumericField[int16](raw, f.Scale)
		case Uint16Type:
			value, err = readNumericField[uint16](raw, f.Scale)
		case Int32Type:
			value, err = readNumericField[int32](raw, f.Scale)
		case Uint32Type:
			value, err = readNumericField[uint32](raw, f.Scale)
		case Float32Type:
			value, err = readNumericField[float32](raw, f.Scale)
		case StringType:
			// ASCII and UTF-8 values
			str := string(raw)
			value = strings.TrimRight(str, "\x00")
		}
		if err != nil {
			return nil, err
		}
		result[f.Name] = createEntry(value, f.Unit)
	}

	return result, nil
}

func buildCoilValues(dataMap map[string][]byte, fields []DataField) (map[string]any, error) {
	result := make(map[string]any)

	for _, f := range fields {
		raw := dataMap[f.Name]
		if len(raw) == 0 {
			continue
		}
		// Modbus packs coils LSB-first: the addressed coil is always in bit 0 of raw[0].
		bit := (raw[0] & 0x01) != 0

		result[f.Name] = bit
	}

	return result, nil
}

func createEntry(value any, unit string) map[string]any {
	entry := map[string]any{"value": value}
	if unit != "" {
		entry["unit"] = unit
	}
	return entry
}
