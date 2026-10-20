package modbus

import "slices"

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
	blockEnd := fields[0].Address + fields[0].Length

	for _, field := range fields[1:] {
		// distance from start of block to end of current field
		distance := int(field.Address + field.Length - 1 - blockStart)

		if distance < maxLen {
			if field.Address+field.Length > blockEnd {
				blockEnd = field.Address + field.Length
			}
			continue
		}

		// close current block
		blocks = append(blocks, Block{
			Start:  blockStart,
			Length: blockEnd - blockStart,
		})

		// start new block
		blockStart = field.Address
		blockEnd = field.Address + field.Length
	}

	// add last block
	blocks = append(blocks, Block{
		Start:  blockStart,
		Length: blockEnd - blockStart,
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
		if field.Address >= block.Start && field.Address < block.Start+block.Length {
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
