package modbus

import (
	"fmt"
	"slices"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// addressSpace is the number of addresses in a Modbus table.
const addressSpace = 1 << 16

type Block struct {
	Start  uint16
	Length uint16
}

func createBlocks(fields []DataField, maxLen int) []Block {
	if len(fields) == 0 {
		return nil
	}

	slices.SortFunc(fields, func(a, b DataField) int {
		return int(a.Address) - int(b.Address)
	})

	var blocks []Block
	blockStart := fields[0].Address
	// The block end is kept as int because a field ending at the last address wraps to 0 in uint16.
	blockEnd := int(fields[0].Address) + int(fields[0].Length)

	for _, field := range fields[1:] {
		// distance from start of block to end of current field
		distance := int(field.Address + field.Length - 1 - blockStart)

		if distance < maxLen {
			if end := int(field.Address) + int(field.Length); end > blockEnd {
				blockEnd = end
			}
			continue
		}

		// close current block
		blocks = append(blocks, Block{
			Start:  blockStart,
			Length: uint16(blockEnd - int(blockStart)),
		})

		// start new block
		blockStart = field.Address
		blockEnd = int(field.Address) + int(field.Length)
	}

	// add last block
	blocks = append(blocks, Block{
		Start:  blockStart,
		Length: uint16(blockEnd - int(blockStart)),
	})

	return blocks
}

// createWriteBlocks groups adjacent fields only, because a write request spanning a gap
// overwrites the addresses between fields.
func createWriteBlocks(fields []DataField, maxLen int) []Block {
	if len(fields) == 0 {
		return nil
	}

	slices.SortFunc(fields, func(a, b DataField) int {
		return int(a.Address) - int(b.Address)
	})

	var blocks []Block
	block := Block{Start: fields[0].Address, Length: fields[0].Length}

	for _, field := range fields[1:] {
		if field.Address == block.Start+block.Length && int(block.Length+field.Length) <= maxLen {
			block.Length += field.Length
			continue
		}

		blocks = append(blocks, block)
		block = Block{Start: field.Address, Length: field.Length}
	}

	return append(blocks, block)
}

// blockFields returns the client's data fields that fall inside the given block.
func blockFields(fields []DataField, block Block) []DataField {
	res := make([]DataField, 0, len(fields))
	for _, field := range fields {
		if field.Address >= block.Start && int(field.Address) < int(block.Start)+int(block.Length) {
			res = append(res, field)
		}
	}
	return res
}

func getBlockMaxLen(funcCode string) (maxLen int) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		maxLen = maxBits
	case WriteCoilsFunc:
		maxLen = maxWriteBits
	default:
		maxLen = maxRegs
	}
	return
}

func calcFieldLengths(fields []DataField) []DataField {
	for i := range fields {
		switch fields[i].Type {
		case Int32Type, Uint32Type, Float32Type:
			fields[i].Length = 2
		case StringType:
			continue
		default:
			fields[i].Length = 1
		}
	}

	return fields
}

// validateFieldAddresses checks that every field ends within the Modbus address space,
// because a block end beyond the last address wraps around in 16-bit arithmetic.
func validateFieldAddresses(fields []DataField) error {
	for _, field := range fields {
		if int(field.Address)+int(field.Length) > addressSpace {
			err := fmt.Errorf(
				"field %s at address %d with length %d",
				field.Name, field.Address, field.Length,
			)
			return errors.Wrap(ErrAddressOutOfRange, err)
		}
	}

	return nil
}
