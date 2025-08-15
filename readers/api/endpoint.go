// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

var header = []string{
	"profile",
	"subtopic",
	"publisher",
	"protocol",
	"name",
	"unit",
	"value",
	"string_value",
	"bool_value",
	"data_value",
	"sum",
	"time",
	"update_time",
}

func listAllMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var page readers.MessagesPage
		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, err
			}
			req.pageMeta.Publisher = pc.PublisherID
		default:
			// Check if user is authorized to read all messages
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}

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

func deleteMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, errors.Wrap(errors.ErrAuthentication, err)
			}
			req.pageMeta.Publisher = pc.PublisherID

		case req.token != "":
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		default:
			return nil, errors.ErrAuthentication
		}

		err := svc.DeleteMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func backupEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
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
		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		messages, err := convertCSVToSenML(req.Messages)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}

		if err := svc.Restore(ctx, messages...); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}

func generateCSV(page readers.MessagesPage) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	if err := convertSenMLToCSV(page, writer); err != nil {
		return nil, err
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func convertCSVToSenML(csvMessages []byte) ([]senml.Message, error) {
	reader := csv.NewReader(bytes.NewReader(csvMessages))

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var messages []senml.Message
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		msg := senml.Message{}
		for i, value := range record {
			if value == "" || i >= len(header) {
				continue
			}

			switch header[i] {
			case "subtopic":
				msg.Subtopic = value
			case "publisher":
				msg.Publisher = value
			case "protocol":
				msg.Protocol = value
			case "name":
				msg.Name = value
			case "unit":
				msg.Unit = value
			case "time":
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					msg.Time = v
				}
			case "update_time":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.UpdateTime = v
				}
			case "value":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.Value = &v
				}
			case "string_value":
				msg.StringValue = &value
			case "data_value":
				msg.DataValue = &value
			case "bool_value":
				if v, err := strconv.ParseBool(value); err == nil {
					msg.BoolValue = &v
				}
			case "sum":
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					msg.Sum = &v
				}
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func convertSenMLToCSV(page readers.MessagesPage, writer *csv.Writer) error {
	for _, msg := range page.Messages {
		if m, ok := msg.(senml.Message); ok {
			row := []string{
				"",
				m.Subtopic,
				m.Publisher,
				m.Protocol,
				m.Name,
				m.Unit,
				getValue(m.Value, ""),
				getValue(m.StringValue, ""),
				getValue(m.BoolValue, ""),
				getValue(m.DataValue, ""),
				getValue(m.Sum, ""),
				fmt.Sprintf("%v", m.Time),
				fmt.Sprintf("%v", m.UpdateTime),
			}

			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}
	return nil
}

func getValue(ptr interface{}, defaultValue string) string {
	switch v := ptr.(type) {
	case *string:
		if v != nil {
			return *v
		}
	case *float64:
		if v != nil {
			return fmt.Sprintf("%v", *v)
		}
	case *bool:
		if v != nil {
			return fmt.Sprintf("%v", *v)
		}
	}
	return defaultValue
}
