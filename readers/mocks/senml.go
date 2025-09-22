// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package mocks

import (
	"context"
	"encoding/json"
	"sync"

	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
)

const noLimit = 0

var _ readers.SenMLMessageRepository = (*senmlRepositoryMock)(nil)

type senmlRepositoryMock struct {
	mu       sync.Mutex
	messages map[string][]readers.Message
}

func NewSenMLRepository(profileID string, messages []readers.Message) readers.SenMLMessageRepository {
	repo := map[string][]readers.Message{
		profileID: messages,
	}

	return &senmlRepositoryMock{
		mu:       sync.Mutex{},
		messages: repo,
	}
}

func (repo *senmlRepositoryMock) ListMessages(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *senmlRepositoryMock) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	return repo.readAll(rpm)
}

func (repo *senmlRepositoryMock) Restore(ctx context.Context, messages ...readers.Message) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, msg := range messages {
		repo.messages[""] = append(repo.messages[""], msg)
	}
	return nil
}

func (repo *senmlRepositoryMock) DeleteMessages(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	for profileID, messages := range repo.messages {
		var remainingMessages []readers.Message
		for _, m := range messages {
			if !repo.senmlMessageMatchesFilter(m, query, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}
		repo.messages[profileID] = remainingMessages
	}
	return nil
}

func (repo *senmlRepositoryMock) readAllJSON(rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var filteredMessages []readers.Message

	// Read from all profiles
	for _, profileMessages := range repo.messages {
		for _, m := range profileMessages {
			if repo.jsonMessageMatchesFilter(m, query, rpm) {
				switch msg := m.(type) {
				case mfjson.Message:
					filteredMessages = append(filteredMessages, msg)
				case map[string]interface{}:
					jsonMsg := mfjson.Message{
						Created:   repo.getCreatedTime(msg),
						Subtopic:  repo.getStringField(msg, "subtopic"),
						Publisher: repo.getStringField(msg, "publisher"),
						Protocol:  repo.getStringField(msg, "protocol"),
						Payload:   repo.getPayload(msg),
					}
					filteredMessages = append(filteredMessages, jsonMsg)
				default:
					continue
				}
			}
		}
	}

	total := uint64(len(filteredMessages))

	if rpm.Offset >= total {
		return readers.JSONMessagesPage{
			JSONPageMetadata: rpm,
			MessagesPage: readers.MessagesPage{
				Total:    total,
				Messages: []readers.Message{},
			},
		}, nil
	}

	end := rpm.Offset + rpm.Limit
	if end > total || rpm.Limit == noLimit {
		end = total
	}

	return readers.JSONMessagesPage{
		JSONPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: filteredMessages[rpm.Offset:end],
		},
	}, nil
}

func (repo *senmlRepositoryMock) readAll(rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, profileMessages := range repo.messages {
		for _, m := range profileMessages {
			if repo.senmlMessageMatchesFilter(m, query, rpm) {
				msgs = append(msgs, m)
			}
		}
	}

	total := uint64(len(msgs))

	if rpm.Offset >= total {
		return readers.SenMLMessagesPage{
			SenMLPageMetadata: rpm,
			MessagesPage: readers.MessagesPage{
				Total:    total,
				Messages: []readers.Message{},
			},
		}, nil
	}

	end := rpm.Offset + rpm.Limit
	if end > total || rpm.Limit == noLimit {
		end = total
	}

	return readers.SenMLMessagesPage{
		SenMLPageMetadata: rpm,
		MessagesPage: readers.MessagesPage{
			Total:    total,
			Messages: msgs[rpm.Offset:end],
		},
	}, nil
}

func (repo *senmlRepositoryMock) jsonMessageMatchesFilter(msg readers.Message, query map[string]interface{}, rpm readers.JSONPageMetadata) bool {
	switch m := msg.(type) {
	case mfjson.Message:
		return repo.checkJSONMessageFilter(m, query, rpm)
	case map[string]interface{}:
		return repo.checkJSONMapFilter(m, query, rpm)
	default:
		return false
	}
}

func (repo *senmlRepositoryMock) senmlMessageMatchesFilter(msg readers.Message, query map[string]interface{}, rpm readers.SenMLPageMetadata) bool {
	switch m := msg.(type) {
	case senml.Message:
		return repo.checkSenMLMessageFilter(m, query, rpm)
	default:
		return false
	}
}

func (repo *senmlRepositoryMock) checkJSONMessageFilter(jsonMsg mfjson.Message, query map[string]interface{}, rpm readers.JSONPageMetadata) bool {
	if rpm.Subtopic != "" && rpm.Subtopic != jsonMsg.Subtopic {
		return false
	}
	if rpm.Publisher != "" && rpm.Publisher != jsonMsg.Publisher {
		return false
	}
	if rpm.Protocol != "" && rpm.Protocol != jsonMsg.Protocol {
		return false
	}
	if rpm.From != 0 && jsonMsg.Created < rpm.From {
		return false
	}
	if rpm.To != 0 && jsonMsg.Created >= rpm.To {
		return false
	}
	return true
}

func (repo *senmlRepositoryMock) checkJSONMapFilter(jsonMap map[string]interface{}, query map[string]interface{}, rpm readers.JSONPageMetadata) bool {
	if rpm.Subtopic != "" {
		if subtopic, ok := jsonMap["subtopic"].(string); !ok || subtopic != rpm.Subtopic {
			return false
		}
	}

	if rpm.Publisher != "" {
		if publisher, ok := jsonMap["publisher"].(string); !ok || publisher != rpm.Publisher {
			return false
		}
	}

	if rpm.Protocol != "" {
		if protocol, ok := jsonMap["protocol"].(string); !ok || protocol != rpm.Protocol {
			return false
		}
	}

	if rpm.From != 0 {
		created := repo.getCreatedTime(jsonMap)
		if created < rpm.From {
			return false
		}
	}

	if rpm.To != 0 {
		created := repo.getCreatedTime(jsonMap)
		if created >= rpm.To {
			return false
		}
	}

	return true
}

func (repo *senmlRepositoryMock) getCreatedTime(jsonMap map[string]interface{}) int64 {
	if created, ok := jsonMap["created"].(float64); ok {
		return int64(created)
	}
	if created, ok := jsonMap["created"].(int64); ok {
		return created
	}
	return 0
}

func (repo *senmlRepositoryMock) getStringField(jsonMap map[string]interface{}, field string) string {
	if value, ok := jsonMap[field].(string); ok {
		return value
	}
	return ""
}

func (repo *senmlRepositoryMock) getPayload(jsonMap map[string]interface{}) []byte {
	if payload, ok := jsonMap["payload"]; ok {
		switch p := payload.(type) {
		case []byte:
			return p
		case string:
			return []byte(p)
		case map[string]interface{}, []interface{}:
			data, _ := json.Marshal(p)
			return data
		default:
			data, _ := json.Marshal(p)
			return data
		}
	}
	return nil
}

func (repo *senmlRepositoryMock) checkSenMLMessageFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.SenMLPageMetadata) bool {
	if rpm.Subtopic != "" && rpm.Subtopic != senmlMsg.Subtopic {
		return false
	}
	if rpm.Publisher != "" && rpm.Publisher != senmlMsg.Publisher {
		return false
	}
	if rpm.Protocol != "" && rpm.Protocol != senmlMsg.Protocol {
		return false
	}
	if rpm.Name != "" && rpm.Name != senmlMsg.Name {
		return false
	}
	if rpm.From != 0 && senmlMsg.Time < rpm.From {
		return false
	}
	if rpm.To != 0 && senmlMsg.Time >= rpm.To {
		return false
	}

	if !repo.checkSenMLValueFilters(senmlMsg, query, rpm) {
		return false
	}

	return true
}

func (repo *senmlRepositoryMock) checkSenMLValueFilters(senmlMsg senml.Message, query map[string]interface{}, rpm readers.SenMLPageMetadata) bool {
	if _, hasValue := query["v"]; hasValue && senmlMsg.Value != nil {
		comparator, hasComparator := query["comparator"]
		if !hasComparator {
			return *senmlMsg.Value == rpm.Value
		}

		switch comparator.(string) {
		case readers.LowerThanKey:
			return *senmlMsg.Value < rpm.Value
		case readers.LowerThanEqualKey:
			return *senmlMsg.Value <= rpm.Value
		case readers.GreaterThanKey:
			return *senmlMsg.Value > rpm.Value
		case readers.GreaterThanEqualKey:
			return *senmlMsg.Value >= rpm.Value
		case readers.EqualKey:
			return *senmlMsg.Value == rpm.Value
		default:
			return *senmlMsg.Value == rpm.Value
		}
	}

	if _, hasBool := query["vb"]; hasBool && senmlMsg.BoolValue != nil {
		return *senmlMsg.BoolValue == rpm.BoolValue
	}

	if _, hasString := query["vs"]; hasString && senmlMsg.StringValue != nil {
		return *senmlMsg.StringValue == rpm.StringValue
	}

	if _, hasData := query["vd"]; hasData && senmlMsg.DataValue != nil {
		return *senmlMsg.DataValue == rpm.DataValue
	}

	if _, hasValue := query["v"]; !hasValue {
		if _, hasBool := query["vb"]; !hasBool {
			if _, hasString := query["vs"]; !hasString {
				if _, hasData := query["vd"]; !hasData {
					return true
				}
			}
		}
	}

	return false
}
