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

// parseCSVFields reads one register per row, matching columns to a header row case-insensitively.
func parseCSVFields(r io.Reader) ([]field, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(ErrInvalidCSVRow, err)
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
			return nil, errors.Wrap(ErrInvalidCSVRow, err)
		}

		f := field{
			Name:      get(row, "name"),
			Type:      get(row, "type"),
			Unit:      get(row, "unit"),
			ByteOrder: get(row, "byte_order"),
		}

		if s := get(row, "scale"); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, errors.Wrap(ErrInvalidCSVRow, errors.New(fmt.Sprintf("line %d: invalid scale", line)))
			}
			f.Scale = v
		}

		if s := get(row, "address"); s != "" {
			v, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return nil, errors.Wrap(ErrInvalidCSVRow, errors.New(fmt.Sprintf("line %d: invalid address", line)))
			}
			f.Address = uint16(v)
		}

		if s := get(row, "length"); s != "" {
			v, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return nil, errors.Wrap(ErrInvalidCSVRow, errors.New(fmt.Sprintf("line %d: invalid length", line)))
			}
			f.Length = uint16(v)
		}

		fields = append(fields, f)
	}

	return fields, nil
}
