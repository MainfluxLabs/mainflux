// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	colName      = "name"
	colType      = "type"
	colUnit      = "unit"
	colByteOrder = "byte_order"
	colScale     = "scale"
	colAddress   = "address"
	colLength    = "length"
)

// parseCSVFields reads one register per row, matching columns to a header row case-insensitively.
func parseCSVFields(r io.Reader) ([]field, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	col := make(map[string]int, len(header))
	for i, name := range header {
		col[strings.ToLower(strings.TrimSpace(name))] = i
	}

	get := func(row []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	var fields []field
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		f := field{
			Name:      get(row, colName),
			Type:      get(row, colType),
			Unit:      get(row, colUnit),
			ByteOrder: get(row, colByteOrder),
		}

		if s := get(row, colScale); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("line %d: invalid %s", line, colScale))
			}
			f.Scale = v
		}

		if s := get(row, colAddress); s != "" {
			v, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("line %d: invalid %s", line, colAddress))
			}
			addr := uint16(v)
			f.Address = &addr
		}

		if s := get(row, colLength); s != "" {
			v, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("line %d: invalid %s", line, colLength))
			}
			f.Length = uint16(v)
		}

		fields = append(fields, f)
	}

	return fields, nil
}
