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

var _ readers.MessageRepository = (*messageRepositoryMock)(nil)

type messageRepositoryMock struct {
	mutex    sync.Mutex
	messages map[string][]readers.Message
}

// NewMessageRepository returns mock implementation of message repository.
func NewMessageRepository(profileID string, messages []readers.Message) readers.MessageRepository {
	repo := map[string][]readers.Message{
		profileID: messages,
	}

	return &messageRepositoryMock{
		mutex:    sync.Mutex{},
		messages: repo,
	}
}

func (repo *messageRepositoryMock) ListProfileMessages(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll(profileID, rpm)
}

func (repo *messageRepositoryMock) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	return repo.readAll("", rpm)
}

func (repo *messageRepositoryMock) Restore(ctx context.Context, format string, messages ...readers.Message) error {
	panic("not implemented")
}

func (repo *messageRepositoryMock) readAll(profileID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if rpm.Format != "" && rpm.Format != "messages" && rpm.Format != "json" {
		return readers.MessagesPage{}, nil
	}

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, m := range repo.messages[profileID] {
		if repo.messageMatchesFilter(m, query, rpm) {
			if msgMap, ok := m.(map[string]interface{}); ok && rpm.Format == "json" {
				jsonMsg := repo.mapToJSONMessage(msgMap)
				msgs = append(msgs, jsonMsg)
			} else {
				msgs = append(msgs, m)
			}
		}
	}

	numOfMessages := uint64(len(msgs))

	if rpm.Offset >= numOfMessages {
		return readers.MessagesPage{}, nil
	}
	if rpm.Limit < 0 {
		return readers.MessagesPage{}, nil
	}

	end := rpm.Offset + rpm.Limit
	if end > numOfMessages || rpm.Limit == noLimit {
		end = numOfMessages
	}

	return readers.MessagesPage{
		PageMetadata: rpm,
		Total:        uint64(len(msgs)),
		Messages:     msgs[rpm.Offset:end],
	}, nil
}

func (repo *messageRepositoryMock) mapToJSONMessage(msgMap map[string]interface{}) mfjson.Message {
	msg := mfjson.Message{}
	if created, ok := msgMap["created"].(int64); ok {
		msg.Created = created
	}

	if subtopic, ok := msgMap["subtopic"].(string); ok {
		msg.Subtopic = subtopic
	}

	if publisher, ok := msgMap["publisher"].(string); ok {
		msg.Publisher = publisher
	}

	if protocol, ok := msgMap["protocol"].(string); ok {
		msg.Protocol = protocol
	}

	if payload, ok := msgMap["payload"].(map[string]interface{}); ok {
		payloadBytes, _ := json.Marshal(payload)
		msg.Payload = payloadBytes
	}

	return msg
}

func (repo *messageRepositoryMock) DeleteMessages(ctx context.Context, rpm readers.PageMetadata) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	var query map[string]interface{}
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	for profileID, messages := range repo.messages {
		var remainingMessages []readers.Message

		for _, m := range messages {
			// Keep the messages if they don't match the filter
			if !repo.messageMatchesFilter(m, query, rpm) {
				remainingMessages = append(remainingMessages, m)
			}
		}

		repo.messages[profileID] = remainingMessages
	}

	return nil
}

func (repo *messageRepositoryMock) messageMatchesFilter(msg readers.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch m := msg.(type) {
	case senml.Message:
		return repo.senmlMessageMatchesFilter(m, query, rpm)
	case mfjson.Message:
		return repo.jsonMessageMatchesFilter(m, query, rpm)
	case map[string]interface{}:
		return repo.jsonMapMessageMatchesFilter(m, query, rpm)
	default:
		return false
	}
}

func (repo *messageRepositoryMock) senmlMessageMatchesFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkSenmlFilterCondition(senmlMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) jsonMessageMatchesFilter(jsonMsg mfjson.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkJsonFilterCondition(jsonMsg, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) jsonMapMessageMatchesFilter(jsonMap map[string]interface{}, query map[string]interface{}, rpm readers.PageMetadata) bool {
	for name := range query {
		if !repo.checkJsonMapFilterCondition(jsonMap, name, query, rpm) {
			return false
		}
	}
	return true
}

func (repo *messageRepositoryMock) checkJsonMapFilterCondition(jsonMap map[string]interface{}, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch filterName {
	case "subtopic":
		if subtopic, ok := jsonMap["subtopic"].(string); ok {
			return rpm.Subtopic == subtopic
		}
		return rpm.Subtopic == ""
	case "publisher":
		if publisher, ok := jsonMap["publisher"].(string); ok {
			return rpm.Publisher == publisher
		}
		return rpm.Publisher == ""
	case "protocol":
		if protocol, ok := jsonMap["protocol"].(string); ok {
			return rpm.Protocol == protocol
		}
		return rpm.Protocol == ""
	case "from":
		if created, ok := jsonMap["created"].(float64); ok {
			return int64(created) >= rpm.From
		}
		if created, ok := jsonMap["created"].(int64); ok {
			return created >= rpm.From
		}
		return false
	case "to":
		if created, ok := jsonMap["created"].(float64); ok {
			return int64(created) < rpm.To
		}
		if created, ok := jsonMap["created"].(int64); ok {
			return created < rpm.To
		}
		return false
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkSenmlFilterCondition(senmlMsg senml.Message, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch filterName {
	case "subtopic":
		return rpm.Subtopic == senmlMsg.Subtopic
	case "publisher":
		return rpm.Publisher == senmlMsg.Publisher
	case "name":
		return rpm.Name == senmlMsg.Name
	case "protocol":
		return rpm.Protocol == senmlMsg.Protocol
	case "v":
		return repo.checkSenmlValueFilter(senmlMsg, query, rpm)
	case "vb":
		return repo.checkSenmlBoolValueFilter(senmlMsg, rpm)
	case "vs":
		return repo.checkSenmlStringValueFilter(senmlMsg, rpm)
	case "vd":
		return repo.checkSenmlDataValueFilter(senmlMsg, rpm)
	case "from":
		return senmlMsg.Time >= rpm.From
	case "to":
		return senmlMsg.Time < rpm.To
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkJsonFilterCondition(jsonMsg mfjson.Message, filterName string, query map[string]interface{}, rpm readers.PageMetadata) bool {
	switch filterName {
	case "subtopic":
		return rpm.Subtopic == jsonMsg.Subtopic
	case "publisher":
		return rpm.Publisher == jsonMsg.Publisher
	case "protocol":
		return rpm.Protocol == jsonMsg.Protocol
	case "from":
		return jsonMsg.Created >= rpm.From
	case "to":
		return jsonMsg.Created < rpm.To
	default:
		return true
	}
}

func (repo *messageRepositoryMock) checkSenmlValueFilter(senmlMsg senml.Message, query map[string]interface{}, rpm readers.PageMetadata) bool {
	if senmlMsg.Value == nil {
		return false
	}

	comparator, ok := query["comparator"]
	if !ok {
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

func (repo *messageRepositoryMock) checkSenmlBoolValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.BoolValue != nil && *senmlMsg.BoolValue == rpm.BoolValue
}

func (repo *messageRepositoryMock) checkSenmlStringValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.StringValue != nil && *senmlMsg.StringValue == rpm.StringValue
}

func (repo *messageRepositoryMock) checkSenmlDataValueFilter(senmlMsg senml.Message, rpm readers.PageMetadata) bool {
	return senmlMsg.DataValue != nil && *senmlMsg.DataValue == rpm.DataValue
}
