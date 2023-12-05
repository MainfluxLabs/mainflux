// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"reflect"

	auth "github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
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

		return generateCSV(page)
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

	// Write header row
	header := []string{
		"ID",
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

	// Process each message and write to CSV using reflection
	for _, msg := range page.Messages {
		value := reflect.ValueOf(msg)
		if value.Kind() != reflect.Struct {
			return nil, errors.ErrWrongMessageType
		}

		var row []string
		for i := 0; i < value.NumField(); i++ {
			fieldValue := fmt.Sprintf("%v", value.Field(i).Interface())
			row = append(row, fieldValue)
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
