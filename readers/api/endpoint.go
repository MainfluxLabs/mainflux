// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	auth "github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func ListChannelMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listChannelMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := authorize(ctx, req.token, req.key, req.chanID); err != nil {
			return nil, errors.Wrap(errors.ErrAuthorization, err)
		}

		page, err := svc.ListChannelMessages(req.chanID, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func listAllMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Check if user is authorized to read all messages
		if err := authorizeAdmin(ctx, auth.RootSubject, req.token); err != nil {
			return nil, err
		}

		page, err := svc.ListAllMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func backupEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Check if user is authorized to read all messages
		if err := authorizeAdmin(ctx, auth.RootSubject, req.token); err != nil {
			return nil, err
		}

		page, err := svc.Backup(req.pageMeta)
		if err != nil {
			return nil, err
		}

		csvData, err := generateCSV(page)
		if err != nil {
			return nil, err
		}

		return backupFileRes{file: csvData}, nil
	}
}

func restoreEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Check if user is authorized to read all messages
		if err := authorizeAdmin(ctx, auth.RootSubject, req.token); err != nil {
			return nil, err
		}

		if err := svc.Restore(ctx, req.Messages...); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}

func generateCSV(page readers.MessagesPage) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{
		"Channel",
		"Subtopic",
		"Publisher",
		"Protocol",
		"Name",
		"Unit",
		"Value",
		"String_value",
		"Bool_value",
		"Data_value",
		"Sum",
		"Time",
		"Update_time",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, msg := range page.Messages {
		switch m := msg.(type) {
		// handle SenML messages
		case senml.Message:
			row := []string{
				m.Channel,
				m.Subtopic,
				m.Publisher,
				m.Protocol,
				m.Name,
				m.Unit,
				getFloatValue(m.Value, ""),
				getStringValue(m.StringValue, ""),
				getBoolValue(m.BoolValue, ""),
				getStringValue(m.DataValue, ""),
				getFloatValue(m.Sum, ""),
				fmt.Sprintf("%v", m.Time),
				fmt.Sprintf("%v", m.UpdateTime),
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		// handle JSON messages
		case map[string]interface{}:
			row := make([]string, len(header))
			for key, value := range m {
				if idx := indexOf(header, key); idx != -1 {
					row[idx] = getValue(value, "")
				}
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		default:
			continue
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Helper function to handle the different types of values.
func getValue(value interface{}, defaultValue string) string {
	switch v := value.(type) {
	case *float64:
		return getFloatValue(v, defaultValue)
	case *string:
		return getStringValue(v, defaultValue)
	case *bool:
		return getBoolValue(v, defaultValue)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Helper function to get the index of a string in a slice.
// Returns -1 if the string is not found in the slice.
func indexOf(slice []string, item string) int {
	for i, val := range slice {
		if val == item {
			return i
		}
	}
	return -1
}

func getStringValue(ptr *string, defaultValue string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

func getFloatValue(ptr *float64, defaultValue string) string {
	if ptr != nil {
		return fmt.Sprintf("%v", *ptr)
	}
	return defaultValue
}

func getBoolValue(ptr *bool, defaultValue string) string {
	if ptr != nil {
		return fmt.Sprintf("%v", *ptr)
	}
	return defaultValue
}
